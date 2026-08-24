package templates

const GoHandler = `package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler is the function entry. Stdlib only in MVP.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"runtime": "go",
		"path":    r.URL.Path,
		"method":  r.Method,
		"ts":      time.Now().Format(time.RFC3339),
	})
}
`

const GoWrapper = `package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"

	user "gscf.local/user/handler"
)

type event struct {
	Method  string            ` + "`json:\"method\"`" + `
	Path    string            ` + "`json:\"path\"`" + `
	Query   map[string]string ` + "`json:\"query\"`" + `
	Headers map[string]string ` + "`json:\"headers\"`" + `
	Body    string            ` + "`json:\"body\"`" + `
}

type result struct {
	Status  int               ` + "`json:\"status\"`" + `
	Headers map[string]string ` + "`json:\"headers\"`" + `
	Body    string            ` + "`json:\"body\"`" + `
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail(err)
	}
	var ev event
	if err := json.Unmarshal(raw, &ev); err != nil {
		fail(err)
	}
	if ev.Method == "" {
		ev.Method = http.MethodGet
	}
	if ev.Path == "" {
		ev.Path = "/"
	}
	q := url.Values{}
	for k, v := range ev.Query {
		q.Set(k, v)
	}
	u := ev.Path
	if enc := q.Encode(); enc != "" {
		if strings.Contains(u, "?") {
			u += "&" + enc
		} else {
			u += "?" + enc
		}
	}
	req := httptest.NewRequest(ev.Method, u, strings.NewReader(ev.Body))
	for k, v := range ev.Headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	user.Handler(rec, req)
	res := rec.Result()
	body, _ := io.ReadAll(res.Body)
	headers := map[string]string{}
	for k, vs := range res.Header {
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}
	out := result{Status: res.StatusCode, Headers: headers, Body: string(body)}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	_ = json.NewEncoder(os.Stdout).Encode(result{Status: 500, Headers: map[string]string{}, Body: err.Error()})
	os.Exit(0)
}
`

const NodeHandler = `// Node.js handler. Export a function.
module.exports = async function handler(event) {
  return {
    statusCode: 200,
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      ok: true,
      runtime: "nodejs",
      path: event.path,
      method: event.method,
      ts: new Date().toISOString(),
    }),
  };
};
`

const NodeWrapper = `'use strict';
const fs = require('fs');
const path = require('path');

function loadUser() {
  const target = process.env.GSCF_USER_MODULE || path.join(__dirname, 'handler.js');
  const mod = require(target);
  if (typeof mod === 'function') return mod;
  if (mod && typeof mod.handler === 'function') return mod.handler;
  if (mod && typeof mod.default === 'function') return mod.default;
  throw new Error('handler export not found');
}

async function main() {
  const raw = fs.readFileSync(0, 'utf8');
  let event = {};
  try { event = JSON.parse(raw || '{}'); } catch (e) {
    write({ status: 400, headers: {}, body: 'invalid event json: ' + e.message });
    return;
  }
  try {
    const fn = loadUser();
    const out = await fn(event);
    if (out == null) {
      write({ status: 204, headers: {}, body: '' });
      return;
    }
    if (typeof out === 'string') {
      write({ status: 200, headers: { 'content-type': 'text/plain' }, body: out });
      return;
    }
    const status = out.statusCode || out.status || 200;
    const headers = out.headers || {};
    let body = out.body;
    if (body != null && typeof body !== 'string') {
      body = JSON.stringify(body);
    }
    write({ status, headers, body: body == null ? '' : String(body) });
  } catch (e) {
    write({ status: 500, headers: {}, body: String(e && e.stack ? e.stack : e) });
  }
}

function write(obj) {
  process.stdout.write(JSON.stringify(obj) + '\n');
}

main();
`
