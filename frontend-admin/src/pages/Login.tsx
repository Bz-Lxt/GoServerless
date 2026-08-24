import { FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, setToken } from "../lib/api";
import { Button, Field, inputCls } from "../lib/ui";
import { useToast } from "../lib/toast";

export default function Login() {
  const nav = useNavigate();
  const toast = useToast();
  const [username, setU] = useState("admin");
  const [password, setP] = useState("admin123");
  const [err, setErr] = useState<{ u?: string; p?: string }>({});
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    const next: typeof err = {};
    if (!username.trim()) next.u = "请输入用户名";
    if (!password.trim()) next.p = "请输入密码";
    setErr(next);
    if (next.u || next.p) {
      toast({ kind: "err", text: "请先补全登录信息" });
      return;
    }
    setBusy(true);
    try {
      const data = await api.login(username.trim(), password);
      setToken(data.token);
      toast({ kind: "ok", text: "欢迎回来" });
      nav("/");
    } catch (ex) {
      toast({ kind: "err", text: ex instanceof Error ? ex.message : "登录失败" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-6">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-md rounded-3xl border border-line bg-surface/80 p-8 shadow-card backdrop-blur"
      >
        <div className="mb-8">
          <div className="mb-3 inline-flex rounded-full border border-amber/30 bg-amber/10 px-3 py-1 text-xs font-semibold text-amber">
            MINI AWS LAMBDA
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">GoServerless</h1>
          <p className="mt-2 text-sm text-muted">把一段代码变成一条可以调用的 URL。</p>
        </div>
        <div className="space-y-4">
          <Field label="用户名 *" error={err.u}>
            <input className={inputCls(!!err.u)} value={username} onChange={(e) => setU(e.target.value)} />
          </Field>
          <Field label="密码 *" error={err.p}>
            <input
              type="password"
              className={inputCls(!!err.p)}
              value={password}
              onChange={(e) => setP(e.target.value)}
            />
          </Field>
          <Button type="submit" disabled={busy}>
            {busy ? "登录中…" : "进入控制台"}
          </Button>
        </div>
      </form>
    </div>
  );
}
