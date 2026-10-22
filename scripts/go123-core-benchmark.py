#!/usr/bin/env python3
"""Run genuine standard-library-only core Go sources on a local Go 1.23 toolchain.

This DOES NOT benchmark MySQL, HTTP, notification delivery, or production ingress.
It copies selected, unmodified production files into temporary minimal Go modules,
which allows offline testing even when the full backend dependencies are absent.
"""
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile


ROOT = Path(__file__).resolve().parent.parent
BACKEND = ROOT / '智慧养老手表管理系统' / 'backend' / 'internal'
HEALTH_BENCH = '''package health
import("testing";"time";"encoding/json";"crypto/sha256")
var benchBytes []byte
var benchSQL string
func BenchmarkNormalizeEncodeIdentity(b *testing.B){now:=time.Date(2026,10,20,12,0,0,0,time.UTC); s:=Sample{RecordID:"sample-1234",Source:"watch",SourceDevice:"watch-1",Metric:"heart_rate",Unit:"count/min",Value:Number(72),Start:now,End:now,Version:1};b.ReportAllocs();for i:=0;i<b.N;i++{t,e:=s.Normalize();if e!=nil{b.Fatal(e)};p,e:=json.Marshal(t);if e!=nil{b.Fatal(e)};identity,_:=json.Marshal([]any{int64(456),(*int64)(nil),int64(1),t.Source,t.SourceDevice,t.RecordID,t.Metric});_=sha256.Sum256(identity);benchBytes=p}}
func BenchmarkSQLValues200x14(b *testing.B){b.ReportAllocs();for i:=0;i<b.N;i++{benchSQL=valueRows(200,14)}}
func BenchmarkSQLValues200x4(b *testing.B){b.ReportAllocs();for i:=0;i<b.N;i++{benchSQL=valueRows(200,4)}}
'''


def go(args, work, env):
    cmd = ['go'] + list(args)
    print(' '.join(cmd), flush=True)
    subprocess.run(cmd, cwd=work, env=env, check=True)


def main():
    version = subprocess.check_output(['go', 'version'], text=True)
    if not re.search(r'\bgo1\.23\.', version):
        raise SystemExit('Requires Go 1.23.x; actual: ' + version.strip())
    env = os.environ.copy()
    env.update(GOTOOLCHAIN='local', GOPROXY='off')
    with tempfile.TemporaryDirectory(prefix='suining-go123-') as d:
        for package, names in {
            'health': ['sample.go', 'batch_sql.go', 'sample_test.go',
                       'validation_test.go', 'batch_sql_test.go'],
            'rules': ['evaluate.go', 'evaluate_test.go', 'performance_test.go'],
        }.items():
            folder = Path(d) / package
            folder.mkdir()
            (folder / 'go.mod').write_text('module benchmark-local\n\ngo 1.23.0\n', encoding='utf-8')
            for name in names:
                shutil.copyfile(BACKEND / package / name, folder / name)
            if package == 'health':
                (folder / 'bench_test.go').write_text(HEALTH_BENCH, encoding='utf-8')
            go(['test', '-race', '-count=1', '-run', '^Test', '-timeout=120s'], folder, env)
            go(['test', '-run', '^$', '-bench', '^Benchmark', '-benchtime=1s', '-count=2'], folder, env)
    print('Core Go 1.23 tests complete; NO end-to-end DB throughput claim.', flush=True)


if __name__ == '__main__':
    main()
