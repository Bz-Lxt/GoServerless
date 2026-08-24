import { useEffect, useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { api, clearToken, type Fn } from "../lib/api";
import { StatusPill } from "../lib/ui";
import { useToast } from "../lib/toast";

export default function Shell() {
  const loc = useLocation();
  const nav = useNavigate();
  const toast = useToast();
  const [fns, setFns] = useState<Fn[]>([]);
  const [open, setOpen] = useState(false);

  async function refresh() {
    try {
      setFns(await api.list());
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "加载失败" });
    }
  }

  useEffect(() => {
    void refresh();
    const t = window.setInterval(() => void refresh(), 4000);
    return () => window.clearInterval(t);
  }, [loc.pathname]);

  return (
    <div className="flex min-h-screen w-full">
      <aside
        className={`fixed inset-y-0 left-0 z-20 w-[248px] border-r border-line bg-surface/90 p-4 backdrop-blur transition md:static md:translate-x-0 ${
          open ? "translate-x-0" : "-translate-x-full md:translate-x-0"
        }`}
      >
        <Link to="/" className="mb-6 block">
          <div className="text-xs font-semibold tracking-[0.2em] text-amber">GOSERVERLESS</div>
          <div className="mt-1 text-lg font-semibold">控制台</div>
        </Link>
        <nav className="space-y-1">
          <Link
            to="/"
            className={`block rounded-xl px-3 py-2 text-sm ${loc.pathname === "/" ? "bg-amber/15 text-amber" : "text-muted hover:bg-white/5"}`}
          >
            总览
          </Link>
        </nav>
        <div className="mt-6 text-xs uppercase tracking-wider text-muted">函数</div>
        <div className="mt-2 max-h-[60vh] space-y-1 overflow-auto">
          {fns.length === 0 ? (
            <div className="px-3 py-6 text-xs text-muted">还没有函数</div>
          ) : (
            fns.map((f) => (
              <Link
                key={f.id}
                to={`/fn/${f.name}`}
                className={`flex items-center justify-between rounded-xl px-3 py-2 text-sm ${
                  loc.pathname === `/fn/${f.name}` ? "bg-white/10 text-ink" : "text-muted hover:bg-white/5"
                }`}
              >
                <span className="truncate font-medium">{f.name}</span>
                <StatusPill status={f.status} />
              </Link>
            ))
          )}
        </div>
        <button
          className="mt-6 text-xs text-muted hover:text-rose"
          onClick={() => {
            clearToken();
            nav("/login");
          }}
        >
          退出登录
        </button>
      </aside>
      <div className="flex min-h-screen w-full flex-1 flex-col">
        <header className="flex items-center justify-between border-b border-line px-4 py-3 md:hidden">
          <button onClick={() => setOpen((v) => !v)} className="text-sm text-muted">
            菜单
          </button>
          <span className="text-sm font-semibold">GoServerless</span>
        </header>
        <main className="w-full flex-1 p-4 md:p-8">
          <Outlet context={{ refresh, fns }} />
        </main>
      </div>
    </div>
  );
}
