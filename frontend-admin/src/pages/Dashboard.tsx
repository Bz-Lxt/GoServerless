import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, type Fn, type Runtime } from "../lib/api";
import { Button, Empty, Field, Modal, StatusPill, inputCls } from "../lib/ui";
import { useToast } from "../lib/toast";

export default function Dashboard() {
  const nav = useNavigate();
  const toast = useToast();
  const [fns, setFns] = useState<Fn[] | null>(null);
  const [ov, setOv] = useState<Record<string, number>>({});
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [runtime, setRt] = useState<Runtime>("go");
  const [desc, setDesc] = useState("");
  const [err, setErr] = useState<string>("");

  async function load() {
    try {
      const [list, overview] = await Promise.all([api.list(), api.overview()]);
      setFns(list);
      setOv(overview);
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "加载失败" });
      setFns([]);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  async function create() {
    if (!/^[a-z][a-z0-9-]{2,39}$/.test(name)) {
      setErr("名称需匹配 ^[a-z][a-z0-9-]{2,39}$");
      toast({ kind: "err", text: "函数名不合法" });
      return;
    }
    setErr("");
    try {
      const fn = await api.create({ name, runtime, description: desc });
      toast({ kind: "ok", text: "已创建，正在打开编辑器" });
      setOpen(false);
      nav(`/fn/${fn.name}`);
    } catch (e) {
      toast({ kind: "err", text: e instanceof Error ? e.message : "创建失败" });
    }
  }

  return (
    <div className="w-full space-y-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">总览</h1>
          <p className="mt-1 text-sm text-muted">热池待命，部署与调用分离。编译永远不在请求路径上。</p>
        </div>
        <Button onClick={() => setOpen(true)}>新建函数</Button>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          ["函数", ov.functions ?? 0],
          ["就绪", ov.ready ?? 0],
          ["热池空闲", ov.pool_idle ?? 0],
          ["热池忙碌", ov.pool_active ?? 0],
        ].map(([k, v]) => (
          <div key={String(k)} className="rounded-2xl border border-line bg-surface p-5 shadow-card">
            <div className="text-xs uppercase tracking-wider text-muted">{k}</div>
            <div className="mt-2 text-3xl font-semibold">{v}</div>
          </div>
        ))}
      </div>

      {!fns ? (
        <div className="text-sm text-muted">加载中…</div>
      ) : fns.length === 0 ? (
        <Empty text="还没有函数。点击右上角「新建函数」开始。" />
      ) : (
        <div className="overflow-x-auto rounded-2xl border border-line">
          <table className="w-full min-w-[720px] text-left text-sm">
            <thead className="bg-surface2 text-xs uppercase tracking-wider text-muted">
              <tr>
                <th className="px-4 py-3">名称</th>
                <th>运行时</th>
                <th>状态</th>
                <th>版本</th>
                <th>端点</th>
              </tr>
            </thead>
            <tbody>
              {fns.map((f) => (
                <tr
                  key={f.id}
                  className="cursor-pointer border-t border-line hover:bg-white/5"
                  onClick={() => nav(`/fn/${f.name}`)}
                >
                  <td className="px-4 py-3 font-medium">{f.name}</td>
                  <td>{f.runtime}</td>
                  <td>
                    <StatusPill status={f.status} />
                  </td>
                  <td>v{f.current_version}</td>
                  <td className="max-w-[280px] truncate font-mono text-xs text-muted">{f.endpoint}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {open ? (
        <Modal title="新建函数" onClose={() => setOpen(false)}>
          <div className="space-y-4">
            <Field label="名称 *" error={err}>
              <input className={inputCls(!!err)} value={name} onChange={(e) => setName(e.target.value)} placeholder="hello-go" />
            </Field>
            <Field label="运行时">
              <select className={inputCls()} value={runtime} onChange={(e) => setRt(e.target.value as Runtime)}>
                <option value="go">Go</option>
                <option value="nodejs">JavaScript (Node)</option>
              </select>
            </Field>
            <Field label="描述">
              <input className={inputCls()} value={desc} onChange={(e) => setDesc(e.target.value)} />
            </Field>
            <div className="flex justify-end gap-2">
              <Button kind="ghost" onClick={() => setOpen(false)}>
                取消
              </Button>
              <Button onClick={() => void create()}>创建</Button>
            </div>
          </div>
        </Modal>
      ) : null}
    </div>
  );
}
