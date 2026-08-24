import Editor from "@monaco-editor/react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { api, getToken, type Fn, type Invocation, type Metrics, type Trigger, type Version } from "../lib/api";
import { Button, Field, Modal, StatusPill, inputCls } from "../lib/ui";
import { useToast } from "../lib/toast";

export default function FunctionEditor() {
  const { name = "" } = useParams();
  const nav = useNavigate();
  const toast = useToast();
  const [fn, setFn] = useState<Fn | null>(null);
  const [code, setCode] = useState("");
  const [logs, setLogs] = useState<string[]>([]);
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [invs, setInvs] = useState<Invocation[]>([]);
  const [vers, setVers] = useState<Version[]>([]);
  const [cron, setCron] = useState("");
  const [envText, setEnvText] = useState("");
  const [timeout, setTimeoutSec] = useState(30);
  const [memory, setMemory] = useState(128);
  const [fieldErr, setFieldErr] = useState<string>("");
  const [confirmDel, setConfirmDel] = useState(false);
  const [busy, setBusy] = useState(false);

  async function load() {
    const data = await api.get(name);
    setFn(data.function);
    setCode(data.code);
    setTimeoutSec(data.function.timeout_sec);
    setMemory(data.function.memory_mb);
    setEnvText(
      Object.entries(data.function.env || {})
        .map(([k, v]) => `${k}=${v}`)
        .join("\n"),
    );
    const [m, i, v, t] = await Promise.all([
      api.metrics(name),
      api.invocations(name),
      api.versions(name),
      api.triggers(name),
    ]);
    setMetrics(m);
    setInvs(i);
    setVers(v);
    const c = t.find((x) => x.kind === "cron");
    setCron(c?.cron_expr || "");
  }

  useEffect(() => {
    void load().catch((e) => toast({ kind: "err", text: e.message }));
  }, [name]);

  useEffect(() => {
    const tok = getToken();
    const es = new EventSource(`/api/v1/functions/${name}/logs/stream?token=${encodeURIComponent(tok)}`);
    es.onmessage = (ev) => {
      setLogs((xs) => [...xs.slice(-200), ev.data]);
    };
    return () => es.close();
  }, [name]);

  const lang = fn?.runtime === "go" ? "go" : "javascript";
  const chart = useMemo(
    () =>
      [...invs].reverse().map((x, idx) => ({
        i: idx + 1,
        exec: x.exec_ms,
        wake: x.wakeup_ms,
      })),
    [invs],
  );

  async function deploy() {
    setBusy(true);
    try {
      await api.deploy(name, code);
      toast({ kind: "ok", text: "已提交构建（异步，不阻塞调用路径）" });
      await load();
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "部署失败" });
    } finally {
      setBusy(false);
    }
  }

  async function saveMeta() {
    const env: Record<string, string> = {};
    for (const line of envText.split("\n")) {
      const t = line.trim();
      if (!t) continue;
      const i = t.indexOf("=");
      if (i <= 0) {
        setFieldErr("环境变量每行必须是 KEY=VALUE");
        toast({ kind: "err", text: "环境变量格式错误" });
        return;
      }
      env[t.slice(0, i)] = t.slice(i + 1);
    }
    if (timeout < 1 || timeout > 300) {
      setFieldErr("超时必须 1-300 秒");
      return;
    }
    setFieldErr("");
    try {
      await api.update(name, { timeout_sec: timeout, memory_mb: memory, env });
      const triggers: Trigger[] = [{ kind: "http", enabled: true }];
      if (cron.trim()) triggers.push({ kind: "cron", cron_expr: cron.trim(), enabled: true });
      await api.putTriggers(name, triggers);
      toast({ kind: "ok", text: "配置已保存" });
      await load();
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "保存失败" });
    }
  }

  async function invokeOnce() {
    try {
      const res = await fetch(fn?.endpoint || `/api/v1/run/${name}`, { method: "POST", body: JSON.stringify({ ping: true }) });
      const text = await res.text();
      toast({ kind: res.ok ? "ok" : "err", text: `调用 ${res.status}: ${text.slice(0, 120)}` });
      await load();
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "调用失败" });
    }
  }

  async function rollback(v: number) {
    try {
      await api.rollback(name, v);
      toast({ kind: "ok", text: `已回滚到 v${v} 并重新构建` });
      await load();
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "回滚失败" });
    }
  }

  if (!fn) return <div className="text-sm text-muted">加载函数…</div>;

  return (
    <div className="w-full space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold">{fn.name}</h1>
            <StatusPill status={fn.status} />
          </div>
          <p className="mt-1 font-mono text-xs text-muted">{fn.endpoint}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button kind="ghost" onClick={() => void invokeOnce()}>
            试调用
          </Button>
          <Button kind="ghost" onClick={() => navigator.clipboard.writeText(fn.endpoint)}>
            复制 URL
          </Button>
          <Button disabled={busy} onClick={() => void deploy()}>
            {busy ? "提交中…" : "部署"}
          </Button>
          <Button kind="danger" onClick={() => setConfirmDel(true)}>
            删除
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
        <div className="overflow-hidden rounded-2xl border border-line bg-[#0b101c]">
          <Editor
            height="520px"
            language={lang}
            theme="vs-dark"
            value={code}
            onChange={(v) => setCode(v || "")}
            options={{ minimap: { enabled: false }, fontSize: 13, fontFamily: "JetBrains Mono" }}
          />
        </div>
        <div className="space-y-4 rounded-2xl border border-line bg-surface p-4">
          <Field label="超时 (秒)" error={fieldErr.includes("超时") ? fieldErr : undefined}>
            <input className={inputCls()} type="number" value={timeout} onChange={(e) => setTimeoutSec(Number(e.target.value))} />
          </Field>
          <Field label="内存 (MB)">
            <select className={inputCls()} value={memory} onChange={(e) => setMemory(Number(e.target.value))}>
              {[64, 128, 256, 512].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Cron 表达式（可选）">
            <input className={inputCls()} value={cron} onChange={(e) => setCron(e.target.value)} placeholder="*/5 * * * *" />
          </Field>
          <Field label="环境变量 KEY=VALUE" error={fieldErr.includes("环境") ? fieldErr : undefined}>
            <textarea className={`${inputCls()} min-h-[120px] font-mono`} value={envText} onChange={(e) => setEnvText(e.target.value)} />
          </Field>
          <Button kind="ghost" onClick={() => void saveMeta()}>
            保存配置
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          ["调用次数", metrics?.invocations ?? 0],
          ["平均耗时 ms", (metrics?.avg_exec_ms ?? 0).toFixed(1)],
          ["P95 / P99", `${(metrics?.p95_exec_ms ?? 0).toFixed(0)} / ${(metrics?.p99_exec_ms ?? 0).toFixed(0)}`],
          ["冷启动", metrics?.cold_starts ?? 0],
        ].map(([k, v]) => (
          <div key={String(k)} className="rounded-2xl border border-line bg-surface p-4">
            <div className="text-xs text-muted">{k}</div>
            <div className="mt-1 text-2xl font-semibold">{v}</div>
          </div>
        ))}
      </div>

      <div className="h-56 rounded-2xl border border-line bg-surface p-3">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={chart}>
            <CartesianGrid stroke="#1E2A44" vertical={false} />
            <XAxis dataKey="i" stroke="#8AA0C2" />
            <YAxis stroke="#8AA0C2" />
            <Tooltip />
            <Area type="monotone" dataKey="exec" stroke="#F5A524" fill="rgba(245,165,36,0.15)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <section className="rounded-2xl border border-line bg-[#070b12] p-4">
          <h2 className="mb-2 text-sm font-semibold">实时日志</h2>
          <pre className="max-h-64 overflow-auto font-mono text-xs text-cyan/90">
            {logs.length ? logs.join("\n") : "等待函数输出…"}
          </pre>
        </section>
        <section className="rounded-2xl border border-line bg-surface p-4">
          <h2 className="mb-2 text-sm font-semibold">版本</h2>
          <div className="space-y-2">
            {vers.map((v) => (
              <div key={v.version} className="flex items-center justify-between rounded-xl bg-bg px-3 py-2 text-sm">
                <div>
                  <div className="font-medium">v{v.version}</div>
                  <div className="text-xs text-muted">
                    {v.created_at} · {v.status}
                  </div>
                </div>
                <Button kind="ghost" onClick={() => void rollback(v.version)}>
                  回滚
                </Button>
              </div>
            ))}
          </div>
        </section>
      </div>

      <div className="overflow-x-auto rounded-2xl border border-line">
        <table className="w-full min-w-[800px] text-left text-sm">
          <thead className="bg-surface2 text-xs uppercase text-muted">
            <tr>
              <th className="px-4 py-3">时间</th>
              <th>状态</th>
              <th>冷启动</th>
              <th>唤醒</th>
              <th>执行</th>
              <th>E2E</th>
            </tr>
          </thead>
          <tbody>
            {invs.map((i) => (
              <tr key={i.id} className="border-t border-line">
                <td className="px-4 py-2 font-mono text-xs">{i.created_at}</td>
                <td>{i.status_code}</td>
                <td className={i.cold_start ? "text-violet" : "text-muted"}>{i.cold_start ? "是" : "否"}</td>
                <td>{i.wakeup_ms}ms</td>
                <td>{i.exec_ms}ms</td>
                <td>{i.e2e_ms}ms</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {confirmDel ? (
        <Modal title="删除函数？" onClose={() => setConfirmDel(false)}>
          <p className="mb-4 text-sm text-muted">将级联清理制品、路由与指标，且不可恢复。</p>
          <div className="flex justify-end gap-2">
            <Button kind="ghost" onClick={() => setConfirmDel(false)}>
              取消
            </Button>
            <Button
              kind="danger"
              onClick={async () => {
                try {
                  await api.remove(name);
                  toast({ kind: "ok", text: "已删除" });
                  nav("/");
                } catch (e) {
                  toast({ kind: "err", text: e instanceof Error ? e.message : "删除失败" });
                }
              }}
            >
              确认删除
            </Button>
          </div>
        </Modal>
      ) : null}
    </div>
  );
}
