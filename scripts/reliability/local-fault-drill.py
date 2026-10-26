#!/usr/bin/env python3
"""Real Windows process/DB/network failure drill on disposable loopback services.

Requires a private lab config generated during environment setup, built server.exe
and reliability.exe, PostgreSQL tools, Prometheus/Alertmanager, and cryptography.
Never contacts people; records receiver acceptance separately from human response.
"""
import argparse
import base64
import datetime
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import uuid


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--lab-config", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--smoke",action="store_true",help="validate setup/TLS/ingestion only; not a complete matrix")
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[2]
    lab = Path(args.lab_config).resolve().parent
    cfg = json.loads(Path(args.lab_config).read_text(encoding="utf-8-sig"))
    private = Path(cfg["fastDataDir"]).resolve()/"process-drill"
    expected_parent = Path(tempfile.gettempdir()).resolve()/"suining-reliability-20261026"
    if private.parent != expected_parent:
        raise ValueError("refusing to stop/start DBs outside the explicitly isolated lab root")
    private = private/uuid.uuid4().hex[:12]
    private.mkdir(parents=True)
    output = Path(args.output).resolve()
    output.mkdir(parents=True, exist_ok=False)
    mysql_bin = Path(r"C:\Program Files\MySQL\MySQL Server 8.0\bin")
    pg_bin = lab/"pgsql"/"bin"
    py = sys.executable
    process_flags = subprocess.CREATE_NO_WINDOW if os.name == "nt" else 0
    children = {}
    cases = []
    dbname = "suining_drill_"+uuid.uuid4().hex[:12]
    mysql_port, pg_port = 33372, cfg["fastPgPort"]
    app_url = "http://127.0.0.1:18081"
    receiver_url = "http://127.0.0.1:18082/alerts"
    endpoint = "https://127.0.0.1:19443/gateway"
    state_file = private/(dbname+"-sessions.json")
    gateway_mode = private/"gateway-mode.json"
    tls_cert, tls_key = private/"gateway-ca.pem", private/"gateway-key.pem"
    gateway_token, receipt_secret, oncall_token = (secrets.token_hex(24) for _ in range(3))
    metrics_token, device_secret = secrets.token_hex(24), secrets.token_hex(24)
    data_key = base64.b64encode(secrets.token_bytes(32)).decode()
    env = dict(os.environ)
    for name in list(env):
        if name.upper().startswith(("MYSQL_","PG_","DATA_ENCRYPTION_KEY","SESSION_","DESKTOP_","SUINING_","NOTIFY_","CARE_","S3_","BLOB_","TRUSTED_PROXY_","RATE_LIMIT_","HEALTH_HOT_CACHE_","LAB_")):
            env.pop(name,None)
    for name in ("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"):
        env.pop(name, None)
    env["NO_PROXY"] = "*"
    env.update(MYSQL_PWD=cfg["password"], PGPASSWORD=cfg["password"])
    mysql_data = private/"mysql"
    mysql_log = output/"mysql-server.txt"

    def run(command, *, data=None, timeout=60, environment=None):
        with tempfile.TemporaryFile() as capture:
            result = subprocess.run([str(x) for x in command], input=data, env=environment or env, stdout=capture,stderr=subprocess.STDOUT, timeout=timeout)
            capture.seek(0)
            text = capture.read().decode(encoding="utf-8",errors="replace").replace(cfg["password"],"[REDACTED]")
        with (output/"setup-commands.txt").open("a",encoding="utf-8") as log:
            log.write(Path(str(command[0])).name+" exit="+str(result.returncode)+"\n"+text+"\n")
        if result.returncode:
            raise RuntimeError("lab command failed: "+Path(str(command[0])).name+" "+text[-3000:])
        return text.strip()

    def sql(statement, database=True):
        command = [mysql_bin/"mysql.exe", "--no-defaults", "--connect-timeout=3", "--host=127.0.0.1", "--port="+str(mysql_port), "--user=root", "--batch", "--raw", "--skip-column-names"]
        if database:
            command.append(dbname)
        command += ["-e", statement]
        return run(command, timeout=15)

    def record(name, **details):
        item = {"case": name, "result": "pass", "unix": time.time(), **details}
        cases.append(item)
        (output/"matrix.json").write_text(json.dumps({**manifest, "cases": cases, "complete": False}, ensure_ascii=False, indent=2), encoding="utf-8")
        print(json.dumps({"case": name, "result": "pass"}), flush=True)

    def spawn(name, command, environment=None):
        log = (output/(name+"-process.txt")).open("ab")
        process = subprocess.Popen([str(x) for x in command], cwd=private, env=environment or env, stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT, creationflags=process_flags)
        log.close()
        children[name] = process
        return process

    def kill(name):
        process = children.get(name)
        if process is not None and process.poll() is None:
            process.kill(); process.wait(timeout=10)

    def wait(label, check, timeout=30):
        until = time.monotonic()+timeout
        last = None
        while time.monotonic() < until:
            try:
                last = check()
                if last:
                    return last
            except (OSError, ValueError, RuntimeError):
                pass
            time.sleep(1)
        raise AssertionError("timed out: "+label+"; last="+repr(last))

    def http(path, body=None, actor=None, metrics=False):
        headers = {}
        if body is not None:
            body = json.dumps(body).encode(); headers["Content-Type"] = "application/json"
        if actor is not None:
            account = accounts[actor-1]
            headers.update(Cookie="JSESSIONID="+account["cookie"], **{"X-CSRF-TOKEN": account["csrf"]})
        if metrics:
            headers["Authorization"] = "Bearer "+metrics_token
        request = urllib.request.Request(app_url+path, body, headers=headers)
        try:
            with urllib.request.urlopen(request, timeout=8) as response:
                return response.status, response.read().decode()
        except urllib.error.HTTPError as error:
            return error.code, error.read().decode()

    def status():
        code, body = http("/restful/service/status")
        return json.loads(body) if code == 200 else {}

    def metrics():
        code, body = http("/metrics", metrics=True)
        if code != 200:
            raise AssertionError("authenticated metrics unavailable")
        return body

    def sos(request_id):
        code, body = http("/restful/device/sos", {"deviceNo": "LAB-WATCH", "secret": device_secret, "requestId": request_id})
        data = json.loads(body)
        if code != 200 or data.get("returnCode") != "1111":
            raise AssertionError("SOS did not commit successfully: "+str(code))
        return int(data["incidentId"])

    def incident(incident_id):
        return json.loads(sql("SELECT JSON_OBJECT('state',state,'acknowledged',acknowledged_by,'resolved',resolved_by) FROM care_incident WHERE id="+str(int(incident_id))))

    def notification(incident_id):
        return json.loads(sql("SELECT JSON_OBJECT('id',id,'status',status,'attempts',attempts,'last_error',last_error,'retry_seconds',TIMESTAMPDIFF(SECOND,NOW(),retry_at)) FROM care_notification_outbox WHERE incident_id="+str(int(incident_id))+" AND channel='sms' ORDER BY id LIMIT 1"))

    def alert(name, since, timeout=100):
        def received():
            path = output/"receiver.jsonl"
            if not path.exists():
                return False
            for line in path.read_text(encoding="utf-8").splitlines():
                event = json.loads(line)
                if event["unix"] < since:
                    continue
                for item in event.get("payload", {}).get("alerts", []):
                    if item.get("labels", {}).get("alertname") == name and item.get("status") == "firing":
                        return event
            return False
        return wait("actual receiver delivery of "+name, received, timeout)

    def mode(value):
        temporary = gateway_mode.with_suffix(".tmp")
        temporary.write_text(json.dumps({"mode": value}), encoding="utf-8")
        temporary.replace(gateway_mode)

    def start_mysql():
        spawn("mysql", [mysql_bin/"mysqld.exe", "--no-defaults", "--basedir="+str(mysql_bin.parent), "--datadir="+str(mysql_data), "--port="+str(mysql_port), "--bind-address=127.0.0.1", "--mysqlx=OFF", "--skip-log-bin", "--max-connections=200", "--console"])
        wait("private MySQL startup", lambda: sql("SELECT 1", False) == "1", 30)

    def stop_mysql():
        location = sql("SELECT @@datadir", False)
        if Path(location).resolve() != mysql_data.resolve():
            raise AssertionError("refusing to stop a non-lab MySQL instance")
        run([mysql_bin/"mysqladmin.exe", "--no-defaults", "--host=127.0.0.1", "--port="+str(mysql_port), "--user=root", "shutdown"], timeout=25)

    app_environment = dict(env, ADDR="127.0.0.1:18081", MYSQL_DSN=f"root:{cfg['password']}@tcp(127.0.0.1:{mysql_port})/{dbname}", PG_ANALYTICS_URL=f"postgres://carelab:{cfg['password']}@127.0.0.1:{pg_port}/{dbname}?sslmode=disable", DATA_ENCRYPTION_KEY=data_key, SESSION_BACKEND="database", DESKTOP_TOKEN=secrets.token_hex(32),BLOB_BACKEND="local",METRICS_TOKEN=metrics_token, CARE_RECEIPT_SECRET=receipt_secret, CARE_NOTIFY_WEBHOOK_TOKEN=gateway_token, CARE_NOTIFY_CA_FILE=str(tls_cert), CARE_NOTIFY_WEBHOOK_URL=endpoint,CARE_ESCALATION_PHONE="13800000002", IMAGE_DIR=str(private/"images"), LAB_DEVICE_SECRET=device_secret, LAB_STATE_FILE=str(state_file))
    def start_app(gateway=endpoint):
        app_environment["CARE_NOTIFY_WEBHOOK_URL"] = gateway
        spawn("app", [lab/"server.exe"], app_environment)
        wait("MySQL-backed app readiness", lambda: status().get("databaseReady"), 35)

    source_sha = run(["git", "-C", repo, "rev-parse", "HEAD"])
    dirty = bool(run(["git", "-C", repo, "status", "--porcelain", "--untracked-files=no"]))
    if dirty:
        raise ValueError("commit the candidate before running the process acceptance matrix")
    binary_hash = hashlib.sha256((lab/"server.exe").read_bytes()).hexdigest()
    manifest = {"source_sha": source_sha, "working_tree_dirty": dirty, "binary_sha256": binary_hash, "scope": "local real processes/MySQL/PostgreSQL/TLS gateway/Prometheus/Alertmanager/receiver; synthetic device and actors; no external host or human paging verified", "mysql_port": mysql_port, "pg_port": pg_port}
    (output/"environment.json").write_text(json.dumps(manifest, indent=2), encoding="utf-8")
    pg_stopped = False
    try:
        for port in (mysql_port, 18081, 18082, 19443, 19090, 19093):
            with socket.socket() as probe:
                probe.bind(("127.0.0.1", port))
        if mysql_data.exists():
            raise ValueError("fault drill requires a new private MySQL directory; retain previous evidence and use a fresh lab root")
        run([mysql_bin/"mysqld.exe", "--no-defaults", "--initialize-insecure", "--basedir="+str(mysql_bin.parent), "--datadir="+str(mysql_data), "--console"], timeout=120)
        # Initialize a password only on this freshly created disposable instance.
        bootstrap_env = dict(env); bootstrap_env.pop("MYSQL_PWD", None)
        spawn("mysql", [mysql_bin/"mysqld.exe", "--no-defaults", "--basedir="+str(mysql_bin.parent), "--datadir="+str(mysql_data), "--port="+str(mysql_port), "--bind-address=127.0.0.1", "--mysqlx=OFF", "--skip-log-bin", "--max-connections=200", "--console"], bootstrap_env)
        def initialize_password():
            command = [mysql_bin/"mysql.exe", "--no-defaults", "--connect-timeout=1", "--host=127.0.0.1", "--port="+str(mysql_port), "--user=root", "-e", "ALTER USER 'root'@'localhost' IDENTIFIED BY '"+cfg["password"]+"'"]
            # SQL is passed on stdin below, never in a process command line.
            statement = command.pop(); command.pop()
            run(command, data=statement.encode(), timeout=3, environment=bootstrap_env)
            return True
        wait("initialize isolated DB credentials", initialize_password, 30)
        sql("CREATE DATABASE `"+dbname+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin", False)
        run([pg_bin/"createdb.exe", "-h", "127.0.0.1", "-p", str(pg_port), "-U", "carelab", dbname])
        from cryptography import x509
        from cryptography.hazmat.primitives import hashes, serialization
        from cryptography.hazmat.primitives.asymmetric import ec
        from cryptography.x509.oid import NameOID, ExtendedKeyUsageOID
        key = ec.generate_private_key(ec.SECP256R1())
        subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "Isolated care gateway test CA")])
        now = datetime.datetime.now(datetime.timezone.utc)
        certificate = (x509.CertificateBuilder().subject_name(subject).issuer_name(subject).public_key(key.public_key()).serial_number(x509.random_serial_number()).not_valid_before(now-datetime.timedelta(minutes=5)).not_valid_after(now+datetime.timedelta(days=2)).add_extension(x509.BasicConstraints(ca=True, path_length=None), critical=True).add_extension(x509.SubjectAlternativeName([x509.IPAddress(ipaddress.ip_address("127.0.0.1"))]), critical=False).add_extension(x509.ExtendedKeyUsage([ExtendedKeyUsageOID.SERVER_AUTH]), critical=False).sign(key, hashes.SHA256()))
        tls_cert.write_bytes(certificate.public_bytes(serialization.Encoding.PEM))
        tls_key.write_bytes(key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption()))
        run([lab/"reliability.exe", "--action", "fixture"], environment=app_environment)
        accounts = json.loads(state_file.read_text())
        mode("normal")
        service_cfg = {"app_url": app_url, "gateway_port": 19443, "receiver_port": 18082, "gateway_token": gateway_token, "receipt_secret": receipt_secret, "oncall_token": oncall_token, "gateway_log": str(output/"gateway.jsonl"), "receiver_log": str(output/"receiver.jsonl"), "gateway_db": str(private/"provider.sqlite"), "gateway_mode": str(gateway_mode), "tls_cert": str(tls_cert), "tls_key": str(tls_key)}
        service_file = private/"services.json"
        service_file.write_text(json.dumps(service_cfg), encoding="utf-8")
        services = repo/"scripts"/"reliability"/"local_services.py"
        spawn("receiver", [py, services, "--role", "receiver", "--config", service_file])
        spawn("gateway", [py, services, "--role", "gateway", "--config", service_file])
        time.sleep(2)
        start_app()
        wait("optional analytics ready", lambda: status().get("analytics", {}).get("ready"), 30)
        # Same rules and real for-durations as deployment; only local scrape and
        # receiver grouping intervals are shorter to improve observation granularity.
        (private/"metrics-token").write_text(metrics_token)
        (private/"oncall-token").write_text(oncall_token)
        (private/"oncall-url").write_text(receiver_url)
        prom_config = private/"prometheus.yml"
        prom_config.write_text("global:\n  scrape_interval: 5s\n  evaluation_interval: 5s\nrule_files:\n  - "+json.dumps(str(repo/"deploy"/"alerts.yml"))+"\nalerting:\n  alertmanagers:\n    - static_configs:\n        - targets: ['127.0.0.1:19093']\nscrape_configs:\n  - job_name: suining\n    metrics_path: /metrics\n    authorization:\n      credentials_file: "+json.dumps(str(private/"metrics-token"))+"\n    static_configs:\n      - targets: ['127.0.0.1:18081']\n")
        am_config = private/"alertmanager.yml"
        am_config.write_text("route:\n  receiver: local-test\n  group_by: [alertname]\n  group_wait: 1s\n  group_interval: 5s\n  repeat_interval: 15s\nreceivers:\n  - name: local-test\n    webhook_configs:\n      - url_file: "+json.dumps(str(private/"oncall-url"))+"\n        send_resolved: true\n        http_config:\n          authorization:\n            credentials_file: "+json.dumps(str(private/"oncall-token"))+"\n")
        spawn("alertmanager", [lab/"alertmanager-0.28.1.windows-amd64"/"alertmanager.exe", "--config.file="+str(am_config), "--storage.path="+str(private/"am-data"), "--web.listen-address=127.0.0.1:19093", "--cluster.listen-address="])
        spawn("prometheus", [lab/"prometheus-3.5.0.windows-amd64"/"prometheus.exe", "--config.file="+str(prom_config), "--storage.tsdb.path="+str(private/"prom-data"), "--web.listen-address=127.0.0.1:19090"])
        watch_env = dict(env, WATCHDOG_STATE=str(private/"watchdog.sqlite"), WATCHDOG_PROMETHEUS_URL="http://127.0.0.1:19090/-/ready", WATCHDOG_ALERTMANAGER_URL="http://127.0.0.1:19093/-/ready", ONCALL_WEBHOOK_URL=receiver_url, ONCALL_TOKEN=oncall_token, WATCHDOG_INTERVAL_SECONDS="3")
        spawn("watchdog", [py, repo/"scripts"/"reliability"/"watchdog.py"], watch_env)
        def monitor_ready(port):
            with urllib.request.urlopen("http://127.0.0.1:"+str(port)+"/-/ready",timeout=2) as response:
                return response.status==200
        wait("Alertmanager startup",lambda:monitor_ready(19093),30)
        wait("Prometheus startup",lambda:monitor_ready(19090),30)
        manifest["mysql_durability"]=json.loads(sql("SELECT JSON_OBJECT('version',@@version,'innodb_flush_log_at_trx_commit',@@innodb_flush_log_at_trx_commit,'datadir',@@datadir)"))
        manifest["postgres_version"]=run([pg_bin/"postgres.exe","--version"])
        (output/"environment.json").write_text(json.dumps(manifest,indent=2),encoding="utf-8")
        initial = sos("initial-duplicate-test")
        assert sos("initial-duplicate-test") == initial
        wait("initial signed delivery", lambda: notification(initial)["status"] == "delivered", 20)
        assert incident(initial)["acknowledged"] is None
        record("SOS duplicate + signed receipt", incident_id=initial, notification=notification(initial), incident=incident(initial))
        code, _ = http("/restful/care/transition", {"id": str(initial), "action": "acknowledge", "note": "synthetic local drill"}, actor=1)
        assert code == 200
        ack_unix = time.time()

        mode("disconnect")
        disconnected=sos("provider-accepted-before-network-disconnect")
        wait("signed receipt after actual connection loss",lambda:notification(disconnected)["status"]=="delivered",25)
        assert notification(disconnected)["attempts"]==1
        record("connection severed after provider acceptance",notification=notification(disconnected),incident=incident(disconnected))
        if args.smoke:
            (output/"matrix.json").write_text(json.dumps({**manifest,"cases":cases,"complete":False,"smoke_only":True},indent=2),encoding="utf-8")
            print("SETUP SMOKE PASSED; full fault matrix not executed",flush=True)
            return

        since = time.time(); mode("503")
        run([pg_bin/"pg_ctl.exe", "-D", str(Path(cfg["fastDataDir"])/"postgres"), "stop", "-m", "immediate", "-w"])
        pg_stopped = True
        failing = sos("pg-down-gateway-503")
        wait("503 durable failure", lambda: notification(failing)["last_error"] == "gateway HTTP 503", 20)
        assert status()["databaseReady"] is True
        alert("SuiningOptionalAnalyticsUnavailable", since, 340)
        alert("SuiningCareQueueStalled", since, 40)
        assert incident(failing)["state"] == "escalated" and incident(failing)["acknowledged"] is None
        record("PG down + persistent 5xx + unattended escalation", incident_id=failing, incident=incident(failing), notification=notification(failing), core_ready=status()["databaseReady"])
        run([pg_bin/"pg_ctl.exe", "-D", str(Path(cfg["fastDataDir"])/"postgres"), "-l", str(output/"pg-recovery.txt"), "start", "-w"])
        pg_stopped = False
        wait("PG recovery", lambda: status().get("analytics", {}).get("ready"), 40)
        wait("projection backlog replay", lambda: sql("SELECT count(*) FROM analytics_outbox") == "0", 60)
        record("PG recovery + backlog replay", pending_projection=0)

        mode("429"); throttled = sos("persistent-429")
        wait("429 first attempt", lambda: notification(throttled)["attempts"] == 1 and notification(throttled)["status"] == "failed", 15)
        before = notification(throttled); time.sleep(8)
        assert notification(throttled)["attempts"] == 1
        wait("429 retry after durable backoff", lambda: notification(throttled)["attempts"] == 2, 40)
        record("429 persists and backs off", initial=before, after=notification(throttled))
        kill("gateway"); unreachable = sos("gateway-process-unavailable")
        wait("gateway network failure", lambda: notification(unreachable)["last_error"] == "gateway network failure", 20)
        record("gateway stopped, outbox retained", notification=notification(unreachable), receiver_alive=children["receiver"].poll() is None)
        mode("normal"); spawn("gateway", [py, services, "--role", "gateway", "--config", service_file]); time.sleep(2)
        kill("app"); start_app("https://synthetic-gateway.invalid/gateway")
        dns_job = sos("dns-failure")
        wait("distinct DNS failure", lambda: notification(dns_job)["last_error"] == "gateway dns failure", 25)
        record("DNS distinguished from HTTP", notification=notification(dns_job))
        kill("app"); mode("kill"); start_app()
        killed_job = sos("worker-killed-mid-send")
        wait("atomic worker claim", lambda: notification(killed_job)["status"] == "sending", 60)
        target_id = notification(killed_job)["id"]
        wait("provider acceptance before killing worker",lambda:any(e.get("event")=="provider_accepted" and e.get("notification_id")==target_id for e in [json.loads(line) for line in (output/"gateway.jsonl").read_text().splitlines()]),10)
        kill("app"); mode("normal"); start_app()
        wait("expired lease reclaimed after actual process kill", lambda: notification(killed_job)["status"] == "delivered" and notification(killed_job)["attempts"] >= 2, 160)
        events = [json.loads(line) for line in (output/"gateway.jsonl").read_text().splitlines()]
        accepted = sum(e.get("event") == "provider_accepted" and e.get("notification_id") == target_id for e in events)
        assert accepted == 1
        record("worker killed mid-send + stale lease recovery", notification=notification(killed_job), provider_acceptances=accepted)

        since = time.time(); stop_mysql()
        wait("business API withdraws on actual MySQL loss", lambda: status().get("databaseReady") is False, 20)
        code, _ = http("/restful/device/sos", {"deviceNo": "LAB-WATCH", "secret": device_secret, "requestId": "must-not-fake-success"})
        assert code == 503
        alert("SuiningBusinessDatabaseUnavailable", since, 100)
        record("MySQL down critical + SOS unavailable", sos_http=code, healthz_http=http("/healthz")[0], prometheus_alive=children["prometheus"].poll() is None)
        start_mysql(); wait("MySQL service self-recovery", lambda: status().get("databaseReady"), 30)
        recovered = sos("mysql-recovered")
        record("MySQL recovery", incident_id=recovered, persisted_incident_count=int(sql("SELECT count(*) FROM care_incident")))

        since = time.time()
        held = spawn("heartbeat-lock", [lab/"reliability.exe", "--action", "hold-heartbeat", "--hold", "150s"], app_environment)
        wait("heartbeat row lock", lambda: "HEARTBEAT_ROW_LOCKED" in (output/"heartbeat-lock-process.txt").read_text(errors="replace"), 15)
        alert("SuiningCareWorkerStale", since, 145)
        assert status()["databaseReady"] is True
        record("care heartbeat blocked + stale alert", app_business_ready=True)
        held.wait(timeout=30)
        wait("heartbeat recovered", lambda: "suining_care_worker_last_heartbeat_unixtime" in metrics(), 15)
        wait("acknowledged but unhandled incident escalates after real deadline", lambda: incident(initial)["state"] == "escalated", max(30, int(ack_unix+630-time.time())))
        record("acknowledged incident unattended timeout", incident_id=initial, elapsed_seconds=round(time.time()-ack_unix,1), incident=incident(initial))
        since = time.time(); kill("app")
        alert("SuiningServiceDown", since, 100)
        assert children["prometheus"].poll() is None and children["receiver"].poll() is None
        record("app killed, monitoring delivers independent alert", prometheus_alive=True, receiver_alive=True)
        start_app()
        since = time.time(); kill("prometheus")
        alert("PrometheusUnavailable", since, 35)
        assert children["watchdog"].poll() is None and children["receiver"].poll() is None
        record("Prometheus killed, separate watchdog delivers", watchdog_alive=True, receiver_alive=True)
        result = {**manifest, "cases": cases, "complete": True, "production_gates": ["External host/whole-host outage detection", "Real provider/hardware delivery", "Actual human acknowledgement/response", "Production DR and key custody"]}
        (output/"matrix.json").write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
        print("LOCAL PROCESS MATRIX COMPLETE", flush=True)
    except BaseException as error:
        (output/"matrix.json").write_text(json.dumps({**manifest, "cases": cases, "complete": False, "failure": str(error)}, ensure_ascii=False, indent=2), encoding="utf-8")
        raise
    finally:
        for name in list(children):
            if name != "mysql":
                kill(name)
        if pg_stopped:
            run([pg_bin/"pg_ctl.exe", "-D", str(Path(cfg["fastDataDir"])/"postgres"), "-l", str(output/"pg-recovery.txt"), "start", "-w"])
        try:
            stop_mysql()
        except (OSError, RuntimeError):
            pass


if __name__ == "__main__":
    main()
