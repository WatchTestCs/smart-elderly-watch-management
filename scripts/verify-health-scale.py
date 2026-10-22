#!/usr/bin/env python3
"""Gate a *completed* isolated health-ingest load run against independent DB counts.

This verifier cannot inspect a live database. Evidence counts must be collected
independently from the same run (and not inferred by multiplying HTTP ACKs).
Never substitute an HTTP-only throughput figure for durable ingestion.
"""
import argparse
import json
import sys
from pathlib import Path


def verify(report, evidence, target):
    if report.get('mode') != 'health':
        raise ValueError('health mode report required')
    run = report.get('run_id')
    if not isinstance(run, str) or not run or evidence.get('run_id') != run:
        raise ValueError('run_id mismatch or absent')
    errors = report.get('errors')
    acked = report.get('acknowledged_samples')
    elapsed = report.get('elapsed_seconds')
    if not isinstance(errors, int) or isinstance(errors, bool) or errors != 0:
        raise ValueError('HTTP errors nonzero/unknown')
    if not isinstance(acked, int) or isinstance(acked, bool) or acked <= 0:
        raise ValueError('invalid HTTP ACK count')
    if not isinstance(elapsed, (int, float)) or isinstance(elapsed, bool) or elapsed <= 0:
        raise ValueError('invalid elapsed time')
    for key in ('persisted_samples', 'persisted_changes'):
        v = evidence.get(key)
        if not isinstance(v, int) or isinstance(v, bool) or v < 0:
            raise ValueError(f'invalid or missing {key} count')
    if evidence['persisted_samples'] != acked or evidence['persisted_changes'] != acked:
        raise ValueError('database rows do not match HTTP acknowledgements')
    # DB evidence must be scoped to this run's recordId prefix. This verifier
    # cannot independently prove the SQL predicate or evidence provenance.
    if evidence.get('scope') != f'recordId prefix {run}-':
        raise ValueError('database counts are not explicitly scoped to this run')
    rate = acked / elapsed
    return {'run_id': run, 'durably_reconciled_samples': acked, 'seconds': elapsed,
            'reconciled_samples_per_second': round(rate, 2), 'target': target,
            'passed': rate >= target,
            'limits': 'DB evidence is user-supplied and must be independently verified; simulated watch sources are not real devices.'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--report', type=Path, required=True, help='cluster-loadtest.py JSON report')
    p.add_argument('--db-evidence', type=Path, required=True, help='independent run-scoped counts JSON')
    p.add_argument('--target', type=int, default=200000)
    args = p.parse_args()
    if args.target < 1:
        p.error('target must be positive')
    try:
        outcome = verify(json.loads(args.report.read_text(encoding='utf-8')),
                         json.loads(args.db_evidence.read_text(encoding='utf-8')), args.target)
    except (OSError, ValueError, TypeError, json.JSONDecodeError) as exc:
        print(f'INCONCLUSIVE/FAILED: {exc}', file=sys.stderr)
        return 2
    print(json.dumps(outcome, ensure_ascii=False, indent=2))
    return 0 if outcome['passed'] else 1


if __name__ == '__main__':
    sys.exit(main())
