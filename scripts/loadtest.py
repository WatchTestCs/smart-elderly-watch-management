"""Synthetic health-ingestion load test. Use an isolated database/device.

Set LOADTEST_DEVICE_NO and LOADTEST_DEVICE_SECRET; credentials are never output.
Each successful request writes one uniquely identified sample for SQL verification.
"""
import argparse
import concurrent.futures
import datetime
import http.client
import json
import math
import os
from pathlib import Path
import time
import urllib.parse
import uuid

p = argparse.ArgumentParser(description=__doc__)
p.add_argument("--url", default="http://127.0.0.1:8080")
p.add_argument("--seconds", type=float, default=20)
p.add_argument("--concurrency", type=int, default=8)
p.add_argument("--binding-version", type=int, default=1)
p.add_argument("--output", type=Path)
a = p.parse_args()
if not 0 < a.seconds <= 3600 or not 0 < a.concurrency <= 256:
    p.error("seconds must be 0..3600 and concurrency 1..256")
u = urllib.parse.urlsplit(a.url)
if u.scheme not in ("http", "https") or u.username or u.password or u.query or u.fragment:
    p.error("Use an HTTP(S) base URL without credentials/query/fragment")
device = os.environ["LOADTEST_DEVICE_NO"]
secret = os.environ["LOADTEST_DEVICE_SECRET"]
run = uuid.uuid4().hex
start = time.perf_counter()
end = start + a.seconds


def worker(index):
    connection_type = http.client.HTTPSConnection if u.scheme == "https" else http.client.HTTPConnection
    connection = connection_type(u.hostname, u.port, timeout=15)
    latencies, errors, statuses = [], 0, {}
    iteration = 0
    while time.perf_counter() < end:
        iteration += 1
        now = datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")
        body = json.dumps({"deviceNo": device, "secret": secret, "bindingVersion": a.binding_version,
            "samples": [{"recordId": f"load-{run}-{index}-{iteration}", "source": "loadtest",
            "sourceDevice": device, "metric": "heart_rate", "unit": "count/min", "value": 72,
            "start": now, "end": now, "version": 1}]})
        at = time.perf_counter()
        try:
            connection.request("POST", u.path.rstrip("/") + "/restful/device/healthBatch", body, {"Content-Type": "application/json"})
            response = connection.getresponse()
            payload = json.loads(response.read())
            statuses[str(response.status)] = statuses.get(str(response.status), 0) + 1
            if response.status != 200 or payload.get("returnCode") != "1111":
                errors += 1
        except Exception:
            errors += 1
            connection.close()
            connection = connection_type(u.hostname, u.port, timeout=15)
        latencies.append((time.perf_counter() - at) * 1000)
    connection.close()
    return latencies, errors, statuses


with concurrent.futures.ThreadPoolExecutor(max_workers=a.concurrency) as executor:
    results = list(executor.map(worker, range(a.concurrency)))
elapsed = time.perf_counter() - start
latencies = sorted(v for values, _, _ in results for v in values)
statuses = {}
for _, _, counts in results:
    for key, value in counts.items():
        statuses[key] = statuses.get(key, 0) + value


def percentile(p):
    return round(latencies[max(0, math.ceil(len(latencies)*p)-1)], 3) if latencies else None


result = {"run_id": run, "scenario": "authenticated_health_ingest_one_sample_per_request",
    "concurrency": a.concurrency, "elapsed_seconds": round(elapsed, 3), "requests": len(latencies),
    "errors": sum(e for _, e, _ in results), "http_statuses": statuses,
    "requests_per_second": round(len(latencies)/elapsed, 2),
    "latency_ms": {"p50": percentile(.50), "p95": percentile(.95), "p99": percentile(.99), "max": percentile(1)}}
encoded = json.dumps(result, indent=2) + "\n"
print(encoded, end="")
if a.output:
    a.output.write_text(encoded, encoding="utf-8")
raise SystemExit(1 if result["errors"] or not latencies else 0)
