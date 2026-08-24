#!/usr/bin/env python3
"""API smoke — runs inside compose network. Cost ¥0."""
from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.request

BASE = os.environ.get("API_BASE", "http://backend:8080")
NAME = f"smoke-{int(time.time())}"


def req(method: str, path: str, body=None, token: str | None = None, timeout=20):
    data = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = urllib.request.Request(BASE + path, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(r, timeout=timeout) as resp:
            raw = resp.read().decode()
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            return e.code, json.loads(raw) if raw else {"message": raw}
        except json.JSONDecodeError:
            return e.code, {"message": raw}


def must(cond, msg):
    if not cond:
        raise SystemExit("FAIL: " + msg)
    print("PASS:", msg)


def main():
    st, body = req("GET", "/api/v1/health")
    must(st == 200 and body.get("data", {}).get("status") == "ok", "health")

    st, body = req("POST", "/api/v1/auth/login", {"username": "admin", "password": "admin123"})
    token = body.get("data", {}).get("token")
    must(st == 200 and token, "login")

    code = """package handler
import ("encoding/json"; "net/http")
func Handler(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")
  _ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "from": "smoke"})
}
"""
    st, body = req("POST", "/api/v1/functions", {
        "name": NAME, "runtime": "nodejs",
        "code": "module.exports = async () => ({ statusCode: 200, body: JSON.stringify({ok:true,from:'smoke'}) });",
    }, token)
    must(st == 201, f"create function {body}")

    st, body = req("POST", f"/api/v1/functions/{NAME}/deploy", {"code": "module.exports = async () => ({ statusCode: 200, body: JSON.stringify({ok:true,from:'smoke'}) });"}, token)
    must(st == 200, "deploy accepted")

    ready = False
    for _ in range(40):
        st, body = req("GET", f"/api/v1/functions/{NAME}", token=token)
        status = (body.get("data") or {}).get("function", {}).get("status")
        if status == "READY":
            ready = True
            break
        if status == "FAILED":
            raise SystemExit("FAIL: build failed " + json.dumps(body))
        time.sleep(1)
    must(ready, "function READY")

    st, body = req("POST", f"/api/v1/run/{NAME}", {"ping": True})
    must(st == 200, f"invoke {st} {body}")

    st, body = req("GET", f"/api/v1/functions/{NAME}/metrics", token=token)
    inv = (body.get("data") or {}).get("invocations", 0)
    must(st == 200 and inv >= 1, f"metrics invocations={inv}")

    st, _ = req("DELETE", f"/api/v1/functions/{NAME}", token=token)
    must(st == 200, "delete")
    st, _ = req("POST", f"/api/v1/run/{NAME}", {})
    must(st == 404, "deleted function 404")
    print("ALL SMOKE PASSED")


if __name__ == "__main__":
    main()
