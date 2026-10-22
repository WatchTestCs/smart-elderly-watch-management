#!/usr/bin/env python3
"""Offline Go 1.23 CPU scheduling and memory-allocation verification.

Uses an isolated stdlib-only module containing COPIES of the production code.
Runs a small, reproducible GOMAXPROCS x GOMEMLIMIT comparison. Not a service
SLA, end-to-end benchmark, NUMA measurement, or a concurrency guarantee.
"""
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent.parent
INTERNAL = ROOT / '智慧养老手表管理系统' / 'backend' / 'internal'
FIXTURES = ROOT / 'scripts' / 'fixtures' / 'go123-perf-polish'
CASES = {
    'rules': ('evaluate.go', 'compact.go', 'evidence_fast.go',
              'evaluate_test.go', 'performance_test.go', 'compact_test.go',
              'evidence_fast_test.go', 'evidence_exact_test.go',
              'evidence_buffer_test.go', 'throughput_test.go',
              'reference_evaluate_test.go'),
    'health': ('sample.go', 'sample_fast.go', 'identity.go', 'batch_sql.go',
               'sample_test.go', 'sample_fast_test.go', 'throughput_test.go',
               'validation_test.go', 'identity_test.go', 'batch_sql_test.go'),
}
BENCHES = {
    'rules': ('BenchmarkEvaluateFastParallel', 'BenchmarkEvaluateFastReusableBufferParallel'),
    'health': ('BenchmarkSamplePrepFastParallel',),
}
RE = re.compile(r'^(Benchmark\w+(?:-\d+)?)\s+(\d+)\s+([\d.]+) ns/op\s+(\d+) B/op\s+(\d+) allocs/op', re.M)

def run():
    version = subprocess.check_output(['go', 'version'], text=True).strip()
    if 'go1.23.' not in version:
        raise SystemExit(f'Go 1.23.x required: {version}')
    env_base = {**os.environ, 'GOPROXY': 'off', 'GOTOOLCHAIN': 'local'}
    subprocess.run(['go', 'test', '-race', '-count=1', './internal/runtuning'], cwd=ROOT / '智慧养老手表管理系统' / 'backend', env=env_base, check=True)
    results = []
    with tempfile.TemporaryDirectory(prefix='suining-memory-scheduling-') as tmp:
        for package, files in CASES.items():
            dest = Path(tmp) / package
            dest.mkdir()
            (dest / 'go.mod').write_text('module localbench\n\ngo 1.23.0\n')
            for f in files:
                shutil.copyfile(INTERNAL / package / f, dest / f)
            if package == 'health':
                for f in ('isolation_types_test.go', 'full_bench_test.go'):
                    shutil.copyfile(FIXTURES / f, dest / f)
            env = {**os.environ, 'GOPROXY': 'off', 'GOTOOLCHAIN': 'local'}
            subprocess.run(['go', 'test', '-count=1', '-run', '^Test', '-timeout=120s'], cwd=dest, env=env, check=True, stdout=subprocess.DEVNULL)
            subprocess.run(['go', 'vet', './...'], cwd=dest, env=env, check=True)
            bench_regex = '^(' + '|'.join(BENCHES[package]) + ')$'
            for memory in ('off', '128MiB'):
                for procs in (1, 2, 4, 8):
                    bench_env = {**env, 'GOMAXPROCS': str(procs), 'GOMEMLIMIT': memory}
                    cmd = ['go', 'test', '-run', '^$', '-bench', bench_regex,
                           '-benchmem', '-benchtime=250ms', '-count=2']
                    print(f'{package} {memory} GOMAXPROCS={procs}', flush=True)
                    raw = subprocess.check_output(cmd, cwd=dest, env=bench_env, text=True)
                    matches = list(RE.finditer(raw))
                    if len(matches) != len(BENCHES[package])*2:
                        raise RuntimeError(f'Unexpected output:\n{raw}')
                    for m in matches:
                        label, iters, nano, b, allocs = m.groups()
                        results.append({
                            'package': package, 'benchmark': label.split('-')[0],
                            'gomaxprocs': procs, 'gomemlimit': memory,
                            'iterations': int(iters), 'ns_per_op': float(nano),
                            'ops_per_second': round(1e9 / float(nano)),
                            'bytes_per_op': int(b), 'allocs_per_op': int(allocs),
                        })
    path = ROOT / 'docs/performance/2026-10-20-go123/memory-scheduling-matrix.json'
    path.write_text(json.dumps({'go': version, 'notes': 'offline compute only; ns/op is aggregate inverse throughput, not p99 latency', 'results': results}, indent=2), encoding='utf-8')
    print(f'Wrote {path} ({len(results)} measurements)', flush=True)

if __name__ == '__main__': run()
