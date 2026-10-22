#!/usr/bin/env python3
"""Build/verify CPU-only long-running Go 1.23 performance harness offline.

Copies the actual production stdlib-only health logic; no hand-written mock
encoders or databases, no hidden network dependencies.
"""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import json

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / '智慧养老手表管理系统/backend/internal/health'
FIX = ROOT / 'scripts/fixtures/go123-perf-polish'
PERF = ROOT / 'scripts/fixtures/go123-lowlevel/perfload.go'


def make_module(path):
    h = path / 'health'
    h.mkdir()
    (path / 'go.mod').write_text('module perfbench\n\ngo 1.23.0\n')
    for f in ['sample.go', 'sample_fast.go', 'identity.go',
              'sample_test.go', 'sample_fast_test.go', 'identity_test.go']:
        shutil.copy2(SRC / f, h / f)
    for f in ['full_bench_test.go', 'isolation_types_test.go']:
        shutil.copy2(FIX / f, h / f)
    # The project Owner type lives in store.go (external database dependency).
    # Use its exact field layout but no fake database.
    (h / 'benchmark_adapter.go').write_text('''package health
    import "crypto/sha256"
    import "encoding/hex"
    type Owner struct {OldID int64;DeviceID *int64;Epoch int64}
    func Digest(b []byte) string {h:=sha256.Sum256(b);return hex.EncodeToString(h[:])}
    func MarshalBenchmark(s Sample)([]byte,error){return marshalSampleFast(s)}
    func IdentityBenchmark(s Sample)string{owner:=int64(123);return identityDigest(Owner{OldID:456, DeviceID:&owner, Epoch:1},s)}
    ''')
    for f in ['isolation_types_test.go']:
        (h / f).unlink()
    cmd = path / 'cmd'
    cmd.mkdir()
    shutil.copy2(PERF, cmd / 'main.go')


def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('--duration', default='12s')
    ap.add_argument('--rates', default='0,6000')
    ap.add_argument('--modes', default='baseline')
    ap.add_argument('--gogc', default='100')
    ap.add_argument('--workers', type=int, default=4)
    ap.add_argument('--output', default='docs/performance/2026-10-20-go123/deep-hardware-results.json')
    args=ap.parse_args()
    env={**os.environ,'GOPROXY':'off','GOTOOLCHAIN':'local'}
    if 'go1.23.' not in subprocess.check_output(['go','version'],text=True):
        raise SystemExit('Requires Go 1.23')
    with tempfile.TemporaryDirectory(prefix='suining-lowlevel-') as tmp:
        path=Path(tmp)
        make_module(path)
        subprocess.run(['go','test','-race','./health'],cwd=path,env=env,check=True)
        subprocess.run(['go','vet','./...'],cwd=path,env=env,check=True)
        program=path/'perfload'
        subprocess.run(['go','build','-o',str(program),'./cmd'],cwd=path,env=env,check=True)
        records=[]
        for rate in map(int,args.rates.split(',')):
            for mode in args.modes.split(','):
                print(f'{mode}, rate={rate}, duration={args.duration}',flush=True)
                raw=subprocess.check_output([str(program),'-mode',mode,'-rate',str(rate),'-duration',args.duration,'-workers',str(args.workers),'-gogc',args.gogc],env=env,text=True)
                case = json.loads(raw)
                if not case['pacing_valid']:
                    print(f"WARNING: non-comparable case: pacing_loss={case['pacing_loss_pct']:.2f}%, drops={case['dropped']}", flush=True)
                records.append(case)
        result={'go':subprocess.check_output(['go','version'],text=True).strip(),
                'warning':'Synthetic CPU-only; 10us p99 histogram buckets (overflow at 200ms); 1ms admission cadence; memory peaks sampled every 100ms; not a network/DB SLA',
                'cases':records}
        output=ROOT/args.output
        output.write_text(json.dumps(result,indent=2,ensure_ascii=False),encoding='utf-8')
        print('saved',output,flush=True)

if __name__=='__main__':main()
