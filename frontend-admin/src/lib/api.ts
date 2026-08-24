export type Runtime = "go" | "nodejs";
export type FnStatus = "DRAFT" | "BUILDING" | "READY" | "FAILED";

export interface Fn {
  id: string;
  name: string;
  runtime: Runtime;
  status: FnStatus;
  description: string;
  timeout_sec: number;
  memory_mb: number;
  max_concurrency: number;
  env: Record<string, string>;
  current_version: number;
  endpoint: string;
  created_at: string;
  updated_at: string;
}

export interface Metrics {
  function_name: string;
  invocations: number;
  successes: number;
  failures: number;
  cold_starts: number;
  avg_exec_ms: number;
  p95_exec_ms: number;
  p99_exec_ms: number;
  avg_wakeup_ms: number;
  pool_hits: number;
  pool_misses: number;
  active_slots: number;
  idle_slots: number;
}

export interface Invocation {
  id: string;
  version: number;
  trigger_kind: string;
  status_code: number;
  success: boolean;
  cold_start: boolean;
  wakeup_ms: number;
  exec_ms: number;
  e2e_ms: number;
  error: string;
  logs: string;
  created_at: string;
}

export interface Trigger {
  id?: string;
  kind: "http" | "cron";
  cron_expr?: string;
  enabled: boolean;
}

export interface Version {
  version: number;
  status: string;
  build_log: string;
  created_at: string;
}

const TOKEN_KEY = "gscf.token";

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Content-Type", "application/json");
  const tok = getToken();
  if (tok) headers.set("Authorization", `Bearer ${tok}`);
  const res = await fetch(path, { ...init, headers });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg = body.message || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return body.data as T;
}

export const api = {
  login: (username: string, password: string) =>
    req<{ token: string; username: string }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  logout: () => req("/api/v1/auth/logout", { method: "POST" }),
  me: () => req<{ username: string }>("/api/v1/auth/me"),
  overview: () => req<Record<string, number>>("/api/v1/overview"),
  templates: () => req<Record<string, string>>("/api/v1/templates"),
  list: () => req<Fn[]>("/api/v1/functions"),
  create: (data: Partial<Fn> & { name: string; runtime: Runtime; code?: string }) =>
    req<Fn>("/api/v1/functions", { method: "POST", body: JSON.stringify(data) }),
  get: (name: string) => req<{ function: Fn; code: string }>(`/api/v1/functions/${name}`),
  update: (name: string, data: unknown) =>
    req<Fn>(`/api/v1/functions/${name}`, { method: "PATCH", body: JSON.stringify(data) }),
  remove: (name: string) => req(`/api/v1/functions/${name}`, { method: "DELETE" }),
  deploy: (name: string, code?: string) =>
    req<Fn>(`/api/v1/functions/${name}/deploy`, { method: "POST", body: JSON.stringify({ code }) }),
  rollback: (name: string, version: number) =>
    req<Fn>(`/api/v1/functions/${name}/rollback`, { method: "POST", body: JSON.stringify({ version }) }),
  versions: (name: string) => req<Version[]>(`/api/v1/functions/${name}/versions`),
  triggers: (name: string) => req<Trigger[]>(`/api/v1/functions/${name}/triggers`),
  putTriggers: (name: string, triggers: Trigger[]) =>
    req<Trigger[]>(`/api/v1/functions/${name}/triggers`, { method: "PUT", body: JSON.stringify({ triggers }) }),
  metrics: (name: string) => req<Metrics>(`/api/v1/functions/${name}/metrics`),
  invocations: (name: string) => req<Invocation[]>(`/api/v1/functions/${name}/invocations?limit=50`),
};
