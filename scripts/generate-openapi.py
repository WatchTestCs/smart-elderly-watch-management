"""Deterministic route inventory plus explicit schemas for the incremental API.

Legacy operations with unknown DTO shapes are labelled as such instead of
inventing contracts. --check fails if the committed document drifts from routes.
"""
import copy
import json
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
backend = root / "智慧养老手表管理系统/backend"
folder = backend / "internal/apidoc"
aliases = json.loads((folder / "aliases.json").read_text(encoding="utf-8"))
paths = {}
public = {"/restful/userLogin/login", "/restful/userLogin/session", "/restful/setup", "/restful/desktop/exit", "/restful/inviteCode/checkCode", "/restful/inviteCode/join"}
hardware = {"/restful/device/" + v for v in ("healthBatch", "login", "bind", "heartbeat", "report", "sos", "commandPull", "commandAck", "fencePoint")} | {"/restful/care/receipt"}
fields = {
    "/restful/userLogin/login": {"phone": "string", "password": "string", "otp": "string", "recoveryCode": "string"},
    "/restful/security/factor": {"action": "string", "password": "string", "otp": "string"},
    "/restful/security/loginAudit": {"after": "integer"},
    "/restful/notification/list": {"after": "integer", "familyId": "integer"},
    "/restful/rules/policy": {"deviceId": "integer", "revision": "integer", "policy": "object"},
    "/restful/device/command": {"deviceId": "integer", "action": "string", "requestKey": "string", "ttlSeconds": "integer"},
    "/restful/device/commandProfile": {"deviceId": "integer", "wireType": "string", "protocol": "string", "enabled": "boolean"},
    "/restful/device/commands": {"deviceId": "integer", "after": "integer"},
    "/restful/health/daily": {"oldId": "integer", "source": "string", "sourceDevice": "string", "metric": "string", "semantics": "string", "day": "string", "offsetMinutes": "integer", "days": "integer"},
    "/restful/alert/review": {"recordId": "integer", "kind": "string", "verdict": "string", "reason": "string", "expectedReviewId": "integer"},
    "/restful/alert/reviews": {"entity": "string", "recordId": "integer", "kind": "string", "after": "integer"},
}
for file in sorted((backend / "internal/handlers").glob("*.go")):
    if file.name.endswith("_test.go"):
        continue
    for method, path in re.findall(r'\.HandleFunc\("(GET|POST|PUT|PATCH|DELETE) ([^" ]+)"', file.read_text(encoding="utf-8-sig")):
        operation = {"summary": path.removeprefix("/restful/"), "deprecated": True, "tags": ["legacy"], "responses": {"200": {"description": "Legacy JSON response; payload varies by operation.", "content": {"application/json": {"schema": {"type": "object", "additionalProperties": True}}}}, "400": {"description": "Invalid request"}, "401": {"description": "Session, device credential or MFA required"}, "403": {"description": "Role, duty or CSRF denied"}, "409": {"description": "Version, binding or state conflict"}, "503": {"description": "Required storage/configuration unavailable"}}}
        operation["security"] = [] if path in public | hardware else [{"sessionCookie": []}]
        if method != "GET":
            operation["parameters"] = [] if path in hardware | {"/restful/setup", "/restful/desktop/exit"} else [{"name": "X-CSRF-TOKEN", "in": "header", "required": True, "schema": {"type": "string"}}]
            schema = {"type": "object", "additionalProperties": True}
            if path in fields:
                schema["properties"] = {key: {"type": kind} for key, kind in fields[path].items()}
                schema["additionalProperties"] = path == "/restful/userLogin/login"
            operation["requestBody"] = {"required": True, "content": {"application/json": {"schema": schema}}}
        operation["description"] = f"Implementation: {file.relative_to(root).as_posix()}. " + ("Device body authentication or signed gateway receipt; not anonymous access." if path in hardware else "Existing wire semantics are retained. Legacy DTOs without explicit schemas remain open objects; consult implementation.")
        paths.setdefault(path, {})[method.lower()] = operation
for alias, canonical in aliases.items():
    if canonical not in paths:
        raise ValueError(f"Alias target has no registered route: {canonical}")
    paths[alias] = copy.deepcopy(paths[canonical])
    for operation in paths[alias].values():
        operation["deprecated"] = False
        operation["tags"] = ["v1"]
        operation["description"] += f" Canonical compatibility target: {canonical}. HTTP method and authorization are unchanged."
spec = {"openapi": "3.1.0", "info": {"title": "岁宁后端 API", "version": (backend / "VERSION").read_text(encoding="utf-8").strip(), "description": "Incremental versioned API and legacy route inventory. https://spec.openapis.org/oas/v3.1.0"}, "components": {"securitySchemes": {"sessionCookie": {"type": "apiKey", "in": "cookie", "name": "JSESSIONID"}}}, "paths": dict(sorted(paths.items()))}
data = (json.dumps(spec, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
target = folder / "openapi.json"
if "--check" in sys.argv:
    if not target.exists() or target.read_bytes() != data:
        raise SystemExit("OpenAPI is stale; run scripts/generate-openapi.py")
else:
    target.write_bytes(data)
print(f"OpenAPI: {len(paths)} paths")
