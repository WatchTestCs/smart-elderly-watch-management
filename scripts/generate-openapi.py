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


def route_policies():
    """Read the explicit Go registry used by auth and CSRF middleware.

    Reject unsupported expressions instead of silently inventing a policy. This
    inventory already reads literal HandleFunc registrations from Go source.
    """
    source = (backend / "internal/accesspolicy/policy.go").read_text(encoding="utf-8")
    table = re.search(r"var routes = map\[route\]rule\{\n(.*?)\n\}", source, re.S)
    if table is None:
        raise ValueError("Cannot locate the accesspolicy route registry")
    policies = {}
    for line in table[1].splitlines():
        if not line.strip() or line.lstrip().startswith("//"):
            continue
        entry = re.fullmatch(
            r'\s*\{http\.Method([A-Za-z]+), "([^" ]+)"\}:\s*'
            r'\{access: (Admin|Public|Device|Shared)(, bootstrap: true)?\},\s*', line
        )
        if entry is None:
            raise ValueError(f"Unsupported literal in accesspolicy registry: {line.strip()}")
        method, path, access, bootstrap = entry.groups()
        key = (method.upper(), path)
        if key in policies:
            raise ValueError(f"Duplicate route policy: {key}")
        policies[key] = (access, bool(bootstrap))
    return policies


policies = route_policies()
csrf_exempt = {path for (_, path), (access, bootstrap) in policies.items() if access == "Device" or bootstrap}
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
registered_routes = {}
for file in sorted((backend / "internal/handlers").glob("*.go")):
    if file.name.endswith("_test.go"):
        continue
    for method, path in re.findall(r'\.HandleFunc\("(GET|POST|PUT|PATCH|DELETE) ([^" ]+)"', file.read_text(encoding="utf-8-sig")):
        key = (method, path)
        if key in registered_routes:
            # Database and database-independent routers intentionally mount this
            # same desktop shutdown handler. Do not allow any other collision.
            if key != ("POST", "/restful/desktop/exit") or registered_routes[key] != file.name:
                raise ValueError(f"Conflicting HTTP route registration: {method} {path} ({registered_routes[key]} and {file.name})")
        else:
            registered_routes[key] = file.name
        access = policies.get((method, path), ("Admin", False))[0]
        operation = {"summary": path.removeprefix("/restful/"), "deprecated": True, "tags": ["legacy"], "responses": {"200": {"description": "Legacy JSON response; payload varies by operation.", "content": {"application/json": {"schema": {"type": "object", "additionalProperties": True}}}}, "400": {"description": "Invalid request"}, "401": {"description": "Session, device credential or MFA required"}, "403": {"description": "Role, duty or CSRF denied"}, "409": {"description": "Version, binding or state conflict"}, "503": {"description": "Required storage/configuration unavailable"}}}
        operation["security"] = [] if access in {"Public", "Device"} else [{"sessionCookie": []}]
        if method != "GET":
            operation["parameters"] = [] if path in csrf_exempt else [{"name": "X-CSRF-TOKEN", "in": "header", "required": True, "schema": {"type": "string"}}]
            schema = {"type": "object", "additionalProperties": True}
            if path in fields:
                schema["properties"] = {key: {"type": kind} for key, kind in fields[path].items()}
                schema["additionalProperties"] = path == "/restful/userLogin/login"
            operation["requestBody"] = {"required": True, "content": {"application/json": {"schema": schema}}}
        operation["description"] = f"Implementation: {file.relative_to(root).as_posix()}. " + ("Device body authentication or signed gateway receipt; not anonymous access." if access == "Device" else "Existing wire semantics are retained. Legacy DTOs without explicit schemas remain open objects; consult implementation.")
        paths.setdefault(path, {})[method.lower()] = operation
for alias, canonical in aliases.items():
    if canonical not in paths:
        raise ValueError(f"Alias target has no registered route: {canonical}")
    if alias in paths:
        raise ValueError(f"Versioned alias shadows a real HTTP route: {alias}")
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
