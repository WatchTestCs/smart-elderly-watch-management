"""Compare redo capacities on an isolated MySQL instance using the real signed HTTP path.

Requires MySQL 8.0.30+ and built backend/health_load executables. MYSQL_PWD supplies
an optional test-instance password. The original redo capacity is restored.
Optional --buffers requires MySQL 8.0.31+; its original buffer size is also restored.
Optional --io-waits samples cumulative file I/O and buffer pressure counters.
Pool metrics include the login, warmup and measured-request window. Optional
measured_pool snapshots exclude login and warmup. Row checks include every
acknowledged warmup and measured sample.
"""
import argparse
import base64
import http.cookiejar
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import threading
import time
import urllib.request
import uuid


def options():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mysql", required=True)
    parser.add_argument("--mysql-port", required=True, type=int)
    parser.add_argument("--mysql-user", default="root")
    parser.add_argument("--backend", required=True, type=Path)
    parser.add_argument("--load", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--capacities", default="104857600,536870912")
    parser.add_argument("--buffers", help="comma-separated buffer pool sizes; requires MySQL 8.0.31+")
    parser.add_argument("--repeats", type=int, default=3)
    parser.add_argument("--requests", type=int, default=12)
    parser.add_argument("--warmup", type=int, default=0)
    parser.add_argument("--ready-timeout", type=float, default=20)
    parser.add_argument("--trace", action="store_true")
    parser.add_argument("--lock-waits", action="store_true")
    parser.add_argument("--io-waits", action="store_true")
    parser.add_argument("--workers", default="1,2,4,8")
    parser.add_argument("--batches", default="2,200")
    args = parser.parse_args()
    args.capacities = [int(v) for v in args.capacities.split(",")]
    args.buffers = [int(v) for v in args.buffers.split(",")] if args.buffers else None
    args.workers = [int(v) for v in args.workers.split(",")]
    args.batches = [int(v) for v in args.batches.split(",")]
    if args.repeats < 1 or args.requests < 1 or min(args.workers) < 1 or args.warmup < 0:
        parser.error("repeats, requests and workers must be positive; warmup must be nonnegative")
    if not 0 < args.ready_timeout <= 300:
        parser.error("ready-timeout must be greater than zero and at most 300 seconds")
    if args.buffers and min(args.buffers) <= 0:
        parser.error("buffer pool sizes must be positive")
    if not all(1 <= v <= 200 for v in args.batches):
        parser.error("batch size must be 1..200")
    args.backend = args.backend.resolve(strict=True)
    args.load = args.load.resolve(strict=True)
    args.output = args.output.resolve()
    args.output.mkdir(parents=True, exist_ok=True)
    return args


class Experiment:
    def __init__(self, args):
        self.args = args
        self.mysql = [args.mysql, "--protocol=TCP", "-h", "127.0.0.1", "-P", str(args.mysql_port),
                      "-u", args.mysql_user, "-N", "-B", "--raw", "--unbuffered"]

    def sql(self, query, database=None):
        return subprocess.check_output(self.mysql + ([database] if database else []) + ["-e", query],
                                       text=True, encoding="utf-8", timeout=30).strip()

    def capacity(self, value):
        self.sql(f"SET GLOBAL innodb_redo_log_capacity={value}")
        if int(self.sql("SELECT @@innodb_redo_log_capacity")) != value:
            raise RuntimeError("MySQL adjusted the requested redo capacity; use an accepted byte value")
        for _ in range(600):
            status = self.sql("SELECT VARIABLE_VALUE FROM performance_schema.global_status WHERE VARIABLE_NAME='Innodb_redo_log_capacity_resized';"
                              "SELECT VARIABLE_VALUE FROM performance_schema.global_status WHERE VARIABLE_NAME='Innodb_redo_log_resize_status';").splitlines()
            if status == [str(value), "OK"]:
                return
            time.sleep(.1)
        raise RuntimeError("redo resize has not completed; inspect the existing MySQL process")

    def buffer(self, value):
        self.sql(f"SET GLOBAL innodb_buffer_pool_size={value}")
        if int(self.sql("SELECT @@innodb_buffer_pool_size")) != value:
            raise RuntimeError("MySQL adjusted the requested buffer pool size; use an accepted byte value")
        for _ in range(600):
            status = self.sql("SELECT VARIABLE_VALUE FROM performance_schema.global_status "
                              "WHERE VARIABLE_NAME='Innodb_buffer_pool_resize_status_code'")
            if status == "0":
                return
            time.sleep(.2)
        raise RuntimeError("buffer pool resize has not completed; inspect the existing MySQL process")

    def monitor(self, stop, rows, errors, database):
        names = ["Innodb_redo_log_current_lsn", "Innodb_redo_log_checkpoint_lsn",
                 "Innodb_redo_log_capacity_resized", "Innodb_row_lock_time", "Innodb_log_waits"]
        fields = ["current_lsn", "checkpoint_lsn", "capacity", "row_lock_time", "log_waits"]
        if self.args.io_waits:
            names.extend(["Innodb_buffer_pool_wait_free", "Innodb_data_fsyncs", "Innodb_os_log_fsyncs",
                          "Innodb_buffer_pool_reads", "Innodb_buffer_pool_pages_flushed"])
            fields.extend(["buffer_pool_wait_free", "data_fsyncs", "log_fsyncs", "buffer_pool_reads", "pages_flushed"])
        pairs = [f"'{key}',MAX(IF(VARIABLE_NAME='{name}',VARIABLE_VALUE,NULL))" for key, name in zip(fields, names)]
        if self.args.io_waits:
            pairs.append("'file_io',(SELECT JSON_ARRAYAGG(JSON_OBJECT('event',EVENT_NAME,'count',COUNT_STAR,"
                "'wait_ps',SUM_TIMER_WAIT,'reads',COUNT_READ,'read_bytes',SUM_NUMBER_OF_BYTES_READ,"
                "'writes',COUNT_WRITE,'write_bytes',SUM_NUMBER_OF_BYTES_WRITE,'misc',COUNT_MISC,'misc_wait_ps',SUM_TIMER_MISC)) "
                "FROM performance_schema.file_summary_by_event_name WHERE EVENT_NAME IN "
                "('wait/io/file/innodb/innodb_log_file','wait/io/file/innodb/innodb_data_file',"
                "'wait/io/file/innodb/innodb_dblwr_file','wait/io/file/sql/binlog'))")
        if self.args.lock_waits:
            digests = {}
            for current, thread in [("r", "rt"), ("b", "bt")]:
                text = f"COALESCE({thread}.PROCESSLIST_INFO,{current}.SQL_TEXT)"
                digests[current] = (f"COALESCE({current}.DIGEST_TEXT,IF(LENGTH({text})<@@performance_schema_max_sql_text_length,"
                    f"STATEMENT_DIGEST_TEXT(REPLACE({text},'?','0')),CONCAT('LONG ',SUBSTRING_INDEX(TRIM({text}),' ',1))))")
            pairs.append("'lock_waits',COALESCE((SELECT JSON_ARRAYAGG(JSON_OBJECT("
                f"'table',l.OBJECT_NAME,'index',l.INDEX_NAME,'requesting_digest',{digests['r']},"
                f"'blocking_digest',{digests['b']},'blocking_statement_ms',b.TIMER_WAIT/1000000000,"
                "'blocking_rows_examined',b.ROWS_EXAMINED,'blocking_state',bt.PROCESSLIST_STATE)) FROM performance_schema.data_lock_waits w "
                "JOIN performance_schema.data_locks l ON l.ENGINE=w.ENGINE AND l.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID "
                "LEFT JOIN performance_schema.events_statements_current r ON r.THREAD_ID=w.REQUESTING_THREAD_ID "
                "LEFT JOIN performance_schema.events_statements_current b ON b.THREAD_ID=w.BLOCKING_THREAD_ID "
                "LEFT JOIN performance_schema.threads rt ON rt.THREAD_ID=w.REQUESTING_THREAD_ID "
                "LEFT JOIN performance_schema.threads bt ON bt.THREAD_ID=w.BLOCKING_THREAD_ID "
                f"WHERE l.OBJECT_SCHEMA='{database}'),JSON_ARRAY())")
        query = "SELECT JSON_OBJECT(" + ",".join(pairs) + ") FROM performance_schema.global_status WHERE VARIABLE_NAME IN (" + ",".join("'" + n + "'" for n in names) + ");\n"
        process = subprocess.Popen(self.mysql + [database], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                   text=True, encoding="utf-8")
        try:
            while True:
                process.stdin.write(query)
                process.stdin.flush()
                row = json.loads(process.stdout.readline())
                row["at"] = time.perf_counter()
                row["at_unix_us"] = time.time_ns() // 1000
                rows.append(row)
                if stop.wait(.025):
                    break
        except Exception as error:
            errors.append(str(error))
        finally:
            process.stdin.close()
            process.wait(timeout=5)

    def round(self, number):
        args = self.args
        database = "sim_perf_" + uuid.uuid4().hex[:12]
        self.sql(f"CREATE DATABASE `{database}`")
        process = None
        config = args.output / f"private-load-{number}.json"
        with socket.socket() as reserved:
            reserved.bind(("127.0.0.1", 0))
            port = reserved.getsockname()[1]
        csrf = ""
        client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

        def api(route, body=None, headers=None):
            request = urllib.request.Request(f"http://127.0.0.1:{port}/api/v1/{route}",
                data=None if body is None else json.dumps(body).encode(),
                headers={"Content-Type": "application/json", "X-CSRF-TOKEN": csrf, **(headers or {})})
            with client.open(request, timeout=15) as response:
                result = json.load(response)
            if result.get("returnCode") != "1111":
                raise RuntimeError(f"{route}: invalid success envelope")
            return result

        env = os.environ.copy()
        for key in list(env):
            if key.startswith(("CARE_NOTIFY_", "CARE_RECEIPT_")) or key in ["PG_ANALYTICS_URL", "CARE_ESCALATION_PHONE", "BLOB_BACKEND"]:
                env.pop(key, None)
        token = secrets.token_hex(32)
        metrics_token = secrets.token_hex(32)
        env.update(ADDR=f"127.0.0.1:{port}", MYSQL_DSN=f"{args.mysql_user}@tcp(127.0.0.1:{args.mysql_port})/{database}?parseTime=true",
                   DATA_ENCRYPTION_KEY=base64.b64encode(secrets.token_bytes(32)).decode(), SESSION_BACKEND="database",
                   DESKTOP_TOKEN=token, METRICS_TOKEN=metrics_token, IMAGE_DIR=str(args.output / "images"))

        def pool_metrics():
            request = urllib.request.Request(f"http://127.0.0.1:{port}/metrics",
                headers={"Authorization": "Bearer " + metrics_token})
            with urllib.request.urlopen(request, timeout=5) as response:
                text = response.read().decode("utf-8")
            values = {name: float(value) for line in text.splitlines()
                      if line.startswith("suining_mysql_pool_") for name, value in [line.split()]}
            if "suining_mysql_pool_wait_total" not in values:
                raise RuntimeError("backend does not expose MySQL pool metrics")
            return values

        def monitor_pool(stop, rows, errors):
            try:
                while not stop.is_set():
                    rows.append({"at": time.perf_counter(), **pool_metrics()})
                    stop.wait(.1)
            except Exception as error:
                errors.append(str(error))
        password = env.get("MYSQL_PWD")
        if password:
            env["MYSQL_DSN"] = f"{args.mysql_user}:{password}@tcp(127.0.0.1:{args.mysql_port})/{database}?parseTime=true"
        log = (args.output / f"server-{number}.log").open("w", encoding="utf-8")
        try:
            process = subprocess.Popen([str(args.backend)], cwd=args.output, env=env, stdout=log, stderr=subprocess.STDOUT)
            deadline = time.monotonic() + args.ready_timeout
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    raise RuntimeError("test backend exited; inspect its log")
                try:
                    if api("service/status").get("databaseReady"):
                        break
                except OSError:
                    pass
                time.sleep(.2)
            else:
                raise RuntimeError("test backend did not become database-ready")
            csrf = api("auth/session")["csrfToken"]
            password = "Test!1" + secrets.token_hex(5)
            api("setup", {"name": "性能测试", "phone": "13900000001", "password": password}, {"X-Suining-Desktop": token})
            api("auth/login", {"phone": "13900000001", "password": password})
            csrf = api("auth/session")["csrfToken"]
            api("guardians/create", {"customerName": "性能监护", "customerAge": 40, "customerSex": "男", "customerPhone": "13900000002", "customerAddress": "isolated"})
            customer = api("guardians/list", {"customerName": "", "page": 1, "pageSize": 20})["customers"][0]
            devices = []
            for index in range(max(args.workers)):
                name = f"性能长者{index}"
                api("residents/create", {"name": name, "bothdate": "1950-01-01", "sex": "女", "phone": f"13800000{index:03d}",
                    "room": str(index), "illness": "", "warn": "", "foodLike": "", "foodHate": "", "drag": "", "customerId": customer["customerId"]})
                owner = api("residents/list", {"name": name, "page": 1, "pageSize": 20})["oldPeople"][0]
                device = api("device/create", {"alias": f"perf-{index}", "model": "isolated", "oldId": owner["oldId"]})
                devices.append({"device_no": device["deviceNo"], "secret": device["secret"]})
            results = []
            for batch in args.batches:
                for workers in args.workers:
                    before = tuple(map(int,self.sql("SELECT (SELECT COUNT(*) FROM health_sample),(SELECT COUNT(*) FROM health_change)",database).split("\t")))
                    workload = {"server": f"http://127.0.0.1:{port}", "devices": devices[:workers],
                        "requests_per_device": args.requests, "warmup_per_device": args.warmup,
                        "batch_size": batch, "signed": True, "metrics_token": metrics_token}
                    trace_path = args.output / f"trace-{number}-{batch}-{workers}.json"
                    if args.trace:
                        workload["trace_file"] = str(trace_path)
                    config.write_text(json.dumps(workload), encoding="utf-8")
                    stop, samples, pool_samples, errors = threading.Event(), [], [], []
                    pool_before = pool_metrics()
                    monitor = threading.Thread(target=self.monitor, args=(stop, samples, errors, database))
                    pool_monitor = threading.Thread(target=monitor_pool, args=(stop, pool_samples, errors))
                    monitor.start()
                    pool_monitor.start()
                    try:
                        run = subprocess.run([str(args.load), str(config)], capture_output=True, text=True, encoding="utf-8", timeout=180)
                    finally:
                        stop.set()
                        monitor.join(timeout=10)
                        pool_monitor.join(timeout=10)
                    # Persist raw outcomes before validation so a rejected load
                    # cannot discard the SQL/IO observations needed to diagnose it.
                    stem = args.output / f"load-{number}-{batch}-{workers}"
                    stem.with_suffix(".stdout.txt").write_text(run.stdout, encoding="utf-8")
                    stem.with_suffix(".stderr.txt").write_text(run.stderr, encoding="utf-8")
                    stem.with_suffix(".diagnostic.json").write_text(json.dumps({
                        "load_returncode": run.returncode,
                        "monitor_errors": errors,
                        "monitor_alive": monitor.is_alive(),
                        "pool_monitor_alive": pool_monitor.is_alive(),
                        "checkpoint_metrics": samples,
                        "pool_metrics": pool_samples,
                    }, indent=2), encoding="utf-8")
                    if monitor.is_alive() or pool_monitor.is_alive() or errors:
                        raise RuntimeError("checkpoint monitor did not complete; diagnostics: " + str(stem))
                    if run.returncode:
                        raise RuntimeError("load failed; raw stdout/stderr and diagnostics: " + str(stem))
                    result = json.loads(run.stdout)
                    if args.trace:
                        (args.output / f"load-{number}-{batch}-{workers}.json").write_text(run.stdout, encoding="utf-8")
                        observations = json.loads(trace_path.read_text(encoding="utf-8"))
                        measured = [point for point in observations if not point["warmup"]]
                        if len(measured) != workers * args.requests:
                            raise RuntimeError("trace does not contain every measured request")
                        outside = [point for point in measured
                                   if point["started_us"] < result["measured_start_us"]
                                   or point["started_us"] + point["duration_us"] > result["measured_end_us"]]
                        if outside:
                            raise RuntimeError(f"measurement window excludes {len(outside)} requests: "
                                f"start={result['measured_start_us']}, end={result['measured_end_us']}, first={outside[0]}")
                        result["verified_trace_requests"] = len(measured)
                    pool_after = pool_metrics()
                    if result.get("measured_pool_before") is not None:
                        measured_before, measured_after = result["measured_pool_before"], result["measured_pool_after"]
                        result["measured_pool_wait_count"] = measured_after["suining_mysql_pool_wait_total"]-measured_before["suining_mysql_pool_wait_total"]
                        result["measured_pool_wait_seconds"] = measured_after["suining_mysql_pool_wait_seconds_total"]-measured_before["suining_mysql_pool_wait_seconds_total"]
                    after = tuple(map(int,self.sql("SELECT (SELECT COUNT(*) FROM health_sample),(SELECT COUNT(*) FROM health_change)",database).split("\t")))
                    delta=tuple(a-b for a,b in zip(after,before))
                    acknowledged = result["successful_samples"] + result["warmup_samples"]
                    if delta != (acknowledged, acknowledged):
                        raise RuntimeError("acknowledged samples differ from sample/history rows")
                    result.update(acknowledged_samples=acknowledged, verified_database_rows=delta[0],
                                  verified_history_rows=delta[1], checkpoint_metrics=samples,
                                  pool_before=pool_before, pool_after=pool_after, pool_metrics=pool_samples,
                                  pool_wait_count=pool_after["suining_mysql_pool_wait_total"]-pool_before["suining_mysql_pool_wait_total"],
                                  pool_wait_seconds=pool_after["suining_mysql_pool_wait_seconds_total"]-pool_before["suining_mysql_pool_wait_seconds_total"],
                                  max_checkpoint_age=max(int(s["current_lsn"]) - int(s["checkpoint_lsn"]) for s in samples))
                    results.append(result)
            return results
        finally:
            config.unlink(missing_ok=True)
            if process:
                process.terminate()
                try:
                    process.wait(timeout=12)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            log.close()
            assert database.startswith("sim_perf_") and len(database) == 21
            self.sql(f"DROP DATABASE `{database}`")


def main():
    args = options()
    experiment = Experiment(args)
    original = int(experiment.sql("SELECT @@innodb_redo_log_capacity"))
    original_buffer = int(experiment.sql("SELECT @@innodb_buffer_pool_size"))
    results = []
    try:
        for repeat in range(args.repeats):
            order = args.capacities if repeat % 2 == 0 else list(reversed(args.capacities))
            buffers = (args.buffers if repeat % 2 == 0 else list(reversed(args.buffers))) if args.buffers else [original_buffer]
            for capacity, buffer in ((capacity, buffer) for capacity in order for buffer in buffers):
                experiment.capacity(capacity)
                if args.buffers:
                    experiment.buffer(buffer)
                rows = experiment.round(len(results))
                settings = json.loads(experiment.sql("SELECT JSON_OBJECT('version',@@version,'buffer_pool_size',@@innodb_buffer_pool_size,'redo_capacity',@@innodb_redo_log_capacity,'flush_log_at_trx_commit',@@innodb_flush_log_at_trx_commit,'sync_binlog',@@sync_binlog,"
                    "'io_capacity',@@innodb_io_capacity,'io_capacity_max',@@innodb_io_capacity_max,'flush_method',@@innodb_flush_method,'log_buffer_size',@@innodb_log_buffer_size,"
                    "'doublewrite',@@innodb_doublewrite,'log_bin',@@log_bin,'binlog_group_commit_sync_delay',@@binlog_group_commit_sync_delay)"))
                results.append({"index": len(results), "capacity": capacity, "settings": settings, "rows": rows})
                (args.output / "redo-ab.json").write_text(json.dumps(results, indent=2), encoding="utf-8")
                print(json.dumps({"round": len(results) - 1, "capacity": capacity, "buffer_pool_size": buffer,
                    "cases": [{k: r[k] for k in ["batch_size", "concurrency", "p99_ms", "samples_per_second", "pool_wait_count", "pool_wait_seconds", "measured_pool_wait_count", "measured_pool_wait_seconds", "max_checkpoint_age"] if k in r} for r in rows]}), flush=True)
    finally:
        try:
            experiment.capacity(original)
        finally:
            if args.buffers:
                experiment.buffer(original_buffer)


if __name__ == "__main__":
    main()
