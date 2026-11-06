"""Exercise the compiled executable over verified TLS using a temporary local CA."""
import http.server
import json
import pathlib
import ssl
import subprocess
import sys
import tempfile
import threading

binary = str(pathlib.Path(sys.argv[1]).resolve())
openssl = sys.argv[2] if len(sys.argv) > 2 else "openssl"
requests = []


class Handler(http.server.BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def handle(self):
        try:
            super().handle()
        except ConnectionResetError:
            pass  # TLS 拒绝测试会主动关闭连接。

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        requests.append((self.path, body))
        reply = json.dumps(dict(returnCode="1111", bound=1, oldId=42, bindingVersion=3)).encode()
        self.send_response(200)
        self.send_header("Content-Length", str(len(reply)))
        self.end_headers()
        self.wfile.write(reply)


with tempfile.TemporaryDirectory(prefix="watch-tls-") as directory:
    root = pathlib.Path(directory)
    cert, key = root / "test-ca.pem", root / "test-key.pem"
    subprocess.run([openssl, "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "2",
                    "-keyout", str(key), "-out", str(cert), "-subj", "/CN=localhost",
                    "-addext", "subjectAltName=DNS:localhost,IP:127.0.0.1",
                    "-addext", "basicConstraints=critical,CA:TRUE"],
                   check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.load_cert_chain(cert, key)
    server.socket = context.wrap_socket(server.socket, server_side=True)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    args = [binary, "--once", "--server", f"https://localhost:{server.server_port}",
            "--secret", "test-only-secret", "--scenario", "normal"]
    try:
        rejected = subprocess.run(args, capture_output=True, timeout=20)
        assert rejected.returncode != 0, "untrusted certificate was accepted"
        assert not requests, "credentials were sent before certificate verification"
        trusted = subprocess.run(args + ["--ca-file", str(cert)], capture_output=True, timeout=20)
        assert trusted.returncode == 0, trusted.stderr.decode(errors="replace")
        assert [p for p, _ in requests] == ["/api/v1/device/login", "/api/v1/device/heartbeat", "/api/v1/device/report"]
        print("HTTPS: untrusted certificate rejected; explicit local CA accepted; full report received")
    finally:
        server.shutdown()
        server.server_close()
        worker.join()
