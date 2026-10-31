#!/usr/bin/env python3
"""Synthetic loopback gateway/receiver for the isolated process fault drill.

No SMS, email or human recipient is contacted. TLS trust stays enabled.
"""
import argparse
import hashlib
import hmac
import json
from pathlib import Path
import sqlite3
import socket
import ssl
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import urllib.request


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--role", choices=("gateway", "receiver"), required=True)
    parser.add_argument("--config", required=True)
    args = parser.parse_args()
    cfg = json.loads(Path(args.config).read_text(encoding="utf-8"))
    lock = threading.Lock()
    log_path = Path(cfg[args.role+"_log"])
    def record(event):
        with lock, log_path.open("a", encoding="utf-8") as log:
            log.write(json.dumps({"unix": time.time(), **event}, ensure_ascii=False)+"\n")
            log.flush()
            import os
            os.fsync(log.fileno())
    if args.role == "gateway":
        with sqlite3.connect(cfg["gateway_db"]) as db:
            db.execute("CREATE TABLE IF NOT EXISTS accepted(id INTEGER PRIMARY KEY,pending INTEGER NOT NULL DEFAULT 0)")
        def receipt_worker():
            while True:
                with sqlite3.connect(cfg["gateway_db"], timeout=5) as db:
                    rows = db.execute("SELECT id,pending FROM accepted WHERE pending>0 LIMIT 20").fetchall()
                for job_id, pending in rows:
                    body = json.dumps({"notificationId": job_id, "status": "delivered"}, separators=(",", ":")).encode()
                    stamp = str(int(time.time()))
                    signature = hmac.new(cfg["receipt_secret"].encode(), stamp.encode()+b"."+body, hashlib.sha256).hexdigest()
                    request = urllib.request.Request(cfg["app_url"]+"/api/v1/care/receipt", body, headers={"Content-Type": "application/json", "X-Care-Timestamp": stamp, "X-Care-Signature": signature})
                    try:
                        with urllib.request.urlopen(request, timeout=3) as response:
                            code = response.status
                        record({"event": "signed_receipt", "notification_id": job_id, "http_status": code})
                        if code == 200:
                            with sqlite3.connect(cfg["gateway_db"], timeout=5) as db:
                                db.execute("UPDATE accepted SET pending=max(0,pending-1) WHERE id=?", (job_id,))
                    except OSError:
                        record({"event": "signed_receipt_retry", "notification_id": job_id})
                time.sleep(1)
        threading.Thread(target=receipt_worker, daemon=True).start()

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *args):
            pass
        def do_GET(self):
            self.send_response(200 if self.path == "/healthz" else 404)
            self.end_headers()
        def do_POST(self):
            token = cfg["gateway_token"] if args.role == "gateway" else cfg["oncall_token"]
            if not hmac.compare_digest(self.headers.get("Authorization", ""), "Bearer "+token):
                self.send_response(401); self.end_headers(); return
            length = int(self.headers.get("Content-Length", "0"))
            if length <= 0 or length > 65536:
                self.send_response(413); self.end_headers(); return
            try:
                data = json.loads(self.rfile.read(length))
            except ValueError:
                self.send_response(400); self.end_headers(); return
            if args.role == "receiver":
                record({"event": "receiver_accepted", "human_acknowledged": False, "payload": data})
                self.send_response(200); self.end_headers(); return
            mode = json.loads(Path(cfg["gateway_mode"]).read_text(encoding="utf-8"))["mode"]
            job_id = int(data.get("notificationId", 0))
            if job_id < 1 or self.headers.get("Idempotency-Key") != "care-outbox-"+str(job_id):
                self.send_response(400); self.end_headers(); return
            if mode in ("429", "503"):
                code = int(mode)
                record({"event": "gateway_rejection", "notification_id": job_id, "http_status": code})
                self.send_response(code)
                if code == 429:
                    self.send_header("Retry-After", "3")
                self.end_headers(); return
            with sqlite3.connect(cfg["gateway_db"], timeout=5) as db:
                changed = db.execute("INSERT OR IGNORE INTO accepted(id,pending) VALUES(?,0)", (job_id,)).rowcount
                if mode != "kill":
                    db.execute("UPDATE accepted SET pending=2 WHERE id=?", (job_id,))
            record({"event": "provider_accepted" if changed else "provider_deduplicated", "notification_id": job_id, "mode": mode})
            if mode == "disconnect":
                self.connection.shutdown(socket.SHUT_RDWR)
                self.connection.close()
                return
            if mode == "kill":
                # Persist provider acceptance, omit callbacks and keep the HTTP
                # call open while the controller kills the actual app process.
                time.sleep(20)
            self.send_response(202)
            self.end_headers()

    port = cfg[args.role+"_port"]
    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    if args.role == "gateway":
        context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        context.minimum_version = ssl.TLSVersion.TLSv1_2
        context.load_cert_chain(cfg["tls_cert"], cfg["tls_key"])
        server.socket = context.wrap_socket(server.socket, server_side=True)
    print(args.role+" loopback fixture ready", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
