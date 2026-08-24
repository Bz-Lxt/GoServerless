import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

type Toast = { id: number; kind: "ok" | "err" | "info"; text: string };

const Ctx = createContext<(t: Omit<Toast, "id">) => void>(() => undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<Toast[]>([]);
  const push = useCallback((t: Omit<Toast, "id">) => {
    const id = Date.now() + Math.random();
    setItems((xs) => [...xs, { ...t, id }]);
    window.setTimeout(() => setItems((xs) => xs.filter((x) => x.id !== id)), 5000);
  }, []);
  const value = useMemo(() => push, [push]);
  return (
    <Ctx.Provider value={value}>
      {children}
      <div className="fixed right-4 top-4 z-50 flex w-[min(92vw,360px)] flex-col gap-2">
        {items.map((t) => (
          <div
            key={t.id}
            className={`flex items-start justify-between gap-3 rounded-xl border px-4 py-3 text-sm shadow-card ${
              t.kind === "err"
                ? "border-rose/40 bg-[#2a1220] text-rose"
                : t.kind === "ok"
                  ? "border-cyan/30 bg-[#0d241f] text-cyan"
                  : "border-line bg-surface2 text-ink"
            }`}
          >
            <span>{t.text}</span>
            <button aria-label="关闭" onClick={() => setItems((xs) => xs.filter((x) => x.id !== t.id))}>
              ×
            </button>
          </div>
        ))}
      </div>
    </Ctx.Provider>
  );
}

export function useToast() {
  return useContext(Ctx);
}
