#!/usr/bin/env python3
"""Offline Go 1.23 regression and memory/CPU benchmarks for the real core sources.
Not an HTTP, MySQL, notification gateway or application build test.
"""
from pathlib import Path
import os, shutil, subprocess, tempfile
ROOT=Path(__file__).resolve().parent.parent
INTERNAL=ROOT/'智慧养老手表管理系统'/'backend'/'internal'
FIXTURES=ROOT/'scripts'/'fixtures'/'go123-perf-polish'
CASES={
 'rules':('evaluate.go','compact.go','evidence_fast.go','evaluate_test.go','performance_test.go','compact_test.go','evidence_fast_test.go','evidence_exact_test.go','evidence_buffer_test.go','throughput_test.go','reference_evaluate_test.go'),
 'health':('sample.go','sample_fast.go','identity.go','sample_test.go','sample_fast_test.go','throughput_test.go','validation_test.go','identity_test.go',),
}

def main():
    version=subprocess.check_output(['go','version'],text=True)
    if 'go1.23.' not in version:
        raise SystemExit('Expected Go 1.23.x: '+version)
    env={**os.environ,'GOPROXY':'off','GOTOOLCHAIN':'local'}
    with tempfile.TemporaryDirectory(prefix='go123-perf-polish-') as tmp:
        for package,files in CASES.items():
            path=Path(tmp)/package
            path.mkdir()
            (path/'go.mod').write_text('module localbench\n\ngo 1.23.0\n',encoding='utf8')
            for name in files:
                shutil.copyfile(INTERNAL/package/name,path/name)
            if package=='health':
                for name in ('isolation_types_test.go','full_bench_test.go'):
                    shutil.copyfile(FIXTURES/name,path/name)
            for args in (
                ['test','-race','-count=1','-run','^Test','-timeout=120s'],
                ['vet','./...'],
                ['test','-run','^$','-bench','^(BenchmarkEvaluateLegacyMarshalEvidence|BenchmarkEvaluateCompactIntoFastJSON|BenchmarkEvaluateCompactInto|BenchmarkIdentityLegacy|BenchmarkIdentityOptimized|BenchmarkSamplePrepLegacy|BenchmarkSamplePrepOptimized|BenchmarkSamplePrepFast|BenchmarkSamplePrepFastUnicode)$','-benchmem','-benchtime=300ms','-count=2'],
            ):
                print(f'[{package}] go {" ".join(args)}',flush=True)
                subprocess.run(['go',*args],cwd=path,env={**env,'GOMAXPROCS':'1'},check=True)
    print('Only independent stdlib-backed Go 1.23 core code was compiled.',flush=True)

if __name__=='__main__': main()
