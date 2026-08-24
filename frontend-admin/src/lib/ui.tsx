import type { ReactNode } from "react";

export function Button({
  children,
  onClick,
  kind = "primary",
  disabled,
  type = "button",
}: {
  children: ReactNode;
  onClick?: () => void;
  kind?: "primary" | "ghost" | "danger";
  disabled?: boolean;
  type?: "button" | "submit";
}) {
  const cls =
    kind === "primary"
      ? "bg-amber text-bg shadow-glow hover:brightness-110"
      : kind === "danger"
        ? "bg-rose/15 text-rose border border-rose/40 hover:bg-rose/25"
        : "border border-line bg-surface2 text-ink hover:border-muted";
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className={`inline-flex items-center justify-center gap-2 rounded-[10px] px-4 py-2 text-sm font-semibold transition disabled:opacity-40 ${cls}`}
    >
      {children}
    </button>
  );
}

export function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: ReactNode;
}) {
  return (
    <label className="block space-y-1.5">
      <span className="text-xs font-medium uppercase tracking-wider text-muted">{label}</span>
      {children}
      {error ? <p className="text-xs text-rose">{error}</p> : null}
    </label>
  );
}

export function inputCls(err?: boolean) {
  return `w-full rounded-[10px] border bg-bg px-3 py-2 text-sm text-ink outline-none transition ${
    err ? "border-rose" : "border-line focus:border-amber"
  }`;
}

export function StatusPill({ status }: { status: string }) {
  const map: Record<string, string> = {
    DRAFT: "bg-white/5 text-muted",
    BUILDING: "bg-amber/15 text-amber animate-pulse",
    READY: "bg-cyan/15 text-cyan",
    FAILED: "bg-rose/15 text-rose",
  };
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${map[status] || "bg-white/5 text-muted"}`}>
      {status}
    </span>
  );
}

export function Modal({
  title,
  children,
  onClose,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/60 p-4">
      <div className="w-full max-w-md rounded-2xl border border-line bg-surface p-5 shadow-card">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-base font-semibold">{title}</h3>
          <button onClick={onClose} className="text-muted hover:text-ink">
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

export function Empty({ text }: { text: string }) {
  return (
    <div className="rounded-2xl border border-dashed border-line bg-surface/50 px-6 py-16 text-center text-sm text-muted">
      {text}
    </div>
  );
}
