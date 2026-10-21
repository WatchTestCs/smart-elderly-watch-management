"""Measure real multi-instance HTTP traffic, with credentials in a private config.

Config: urls[], mode (health/announcement), devices[] for health or cookie/csrf
for announcements. Run against an isolated fixture. Reported fanout is expected
until separately reconciled with the database and gateway unique ACK counts.
"""
import argparse
import concurrent.futures
import datetime
import http.client
import json
import math
import multiprocessing
from pathlib import Path
import time
import urllib.parse
import uuid


def generate(process, config, run, start, duration, threads, batch):
    urls = [urllib.parse.urlsplit(url) for url in config['urls']]
    for url in urls:
        if url.scheme not in ('http', 'https') or url.username or url.password or url.query or url.fragment:
            raise ValueError('URLs must not contain credentials/query/fragment')

    def thread(index):
        connections = {}
        latencies, errors, completed = [], 0, 0
        statuses = {}
        sequence = 0
        time.sleep(max(0, start-time.time()))
        while time.time() < start+duration:
            sequence += 1
            identity = f'{run}-{process}-{index}-{sequence}'
            if config['mode'] == 'health':
                device = config['devices'][(process*threads+index+sequence) % len(config['devices'])]
                now = datetime.datetime.now(datetime.timezone.utc).isoformat().replace('+00:00', 'Z')
                body = {'deviceNo': device['deviceNo'], 'secret': device['secret'], 'bindingVersion': device.get('bindingVersion', 1),
                        'samples': [{'recordId': f'{identity}-{i}', 'source': 'cluster-benchmark', 'sourceDevice': device['deviceNo'],
                                     'metric': 'heart_rate', 'unit': 'count/min', 'value': 72, 'start': now, 'end': now, 'version': 1} for i in range(batch)]}
                path = '/restful/device/healthBatch'
                headers = {'Content-Type': 'application/json'}
            else:
                body = {'title': f'cl-{identity}', 'textMain': 'Synthetic cluster benchmark', 'fileListName': []}
                path = '/restful/text/add'
                headers = {'Content-Type': 'application/json', 'Cookie': config['cookie'], 'X-CSRF-TOKEN': config['csrf']}
            body = json.dumps(body, separators=(',', ':'))
            endpoint = (process*threads+index+sequence) % len(urls)
            url = urls[endpoint]
            kind = http.client.HTTPSConnection if url.scheme == 'https' else http.client.HTTPConnection
            if endpoint not in connections:
                connections[endpoint] = kind(url.hostname, url.port, timeout=40)
            at = time.perf_counter()
            try:
                conn = connections[endpoint]
                conn.request('POST', url.path.rstrip('/')+path, body, headers)
                response = conn.getresponse()
                statuses[str(response.status)] = statuses.get(str(response.status), 0)+1
                payload = json.loads(response.read())
                success = response.status == 200 and payload.get('returnCode') == '1111'
                if config['mode'] == 'health':
                    success = success and payload.get('accepted') == batch and payload.get('duplicates') == 0
                if success:
                    completed += 1
                else:
                    errors += 1
            except Exception:
                errors += 1
                connections.pop(endpoint).close()
            latencies.append((time.perf_counter()-at)*1000)
        for connection in connections.values():
            connection.close()
        return latencies, errors, completed, statuses

    with concurrent.futures.ThreadPoolExecutor(max_workers=threads) as executor:
        results = list(executor.map(thread, range(threads)))
    return {'latencies': [t for r in results for t in r[0]], 'errors': sum(r[1] for r in results),
            'commits': sum(r[2] for r in results), 'end': time.time(), 'statuses': [r[3] for r in results]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--processes', type=int, default=4)
    parser.add_argument('--threads', type=int, default=8)
    parser.add_argument('--seconds', type=float, default=20)
    parser.add_argument('--batch-size', type=int, default=200)
    args = parser.parse_args()
    if not (1 <= args.processes <= 32 and 1 <= args.threads <= 64 and 0 < args.seconds <= 3600 and 1 <= args.batch_size <= 200):
        parser.error('invalid process/thread/duration/batch bounds')
    config = json.loads(args.config.read_text(encoding='utf-8'))
    if config.get('mode') not in ('health', 'announcement') or not config.get('urls'):
        parser.error('mode must be health/announcement and urls must not be empty')
    if config['mode'] == 'health' and not config.get('devices'):
        parser.error('health mode requires devices')
    run = uuid.uuid4().hex[:12]
    start = time.time()+2
    with concurrent.futures.ProcessPoolExecutor(max_workers=args.processes) as executor:
        futures = [executor.submit(generate, i, config, run, start, args.seconds, args.threads, args.batch_size) for i in range(args.processes)]
        results = [f.result() for f in futures]
    elapsed = max(r['end'] for r in results)-start
    latencies = sorted(t for r in results for t in r['latencies'])
    def percentile(p):
        return round(latencies[max(0, math.ceil(len(latencies)*p)-1)], 3) if latencies else None
    statuses = {}
    for result in results:
        for group in result['statuses']:
            for code, count in group.items():
                statuses[code] = statuses.get(code, 0)+count
    commits = sum(r['commits'] for r in results)
    value = {'run_id': run, 'mode': config['mode'], 'application_instances': len(config['urls']),
             'client_processes': args.processes, 'concurrency': args.processes*args.threads,
             'started_unix': start, 'ended_unix': start+elapsed, 'elapsed_seconds': round(elapsed, 3),
             'requests': len(latencies), 'http_commit_acknowledgements': commits,
             'errors': sum(r['errors'] for r in results), 'http_statuses': statuses,
             'requests_per_second': round(len(latencies)/elapsed, 2),
             'latency_ms': {'p50': percentile(.5), 'p95': percentile(.95), 'p99': percentile(.99), 'max': percentile(1)}}
    if config['mode'] == 'health':
        value['samples_per_request'] = args.batch_size
        value['acknowledged_samples'] = commits*args.batch_size
        value['acknowledged_samples_per_second'] = round(commits*args.batch_size/elapsed, 2)
    else:
        value['expected_outbox_rows'] = commits*config['recipients']*2
        value['expected_outbox_rows_per_second'] = round(value['expected_outbox_rows']/elapsed, 2)
        value['verification'] = 'Expected fanout only. Reconcile SQL rows and gateway ACKs before claiming throughput.'
    args.output.write_text(json.dumps(value, indent=2)+'\n', encoding='utf-8')
    print(json.dumps(value, indent=2))
    raise SystemExit(1 if value['errors'] or not commits else 0)


if __name__ == '__main__':
    multiprocessing.freeze_support()
    main()
