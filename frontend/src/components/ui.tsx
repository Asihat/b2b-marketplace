import type { ReactNode } from "react";

export function PageHeader({ title, action }: { title: string; action?: ReactNode }) {
  return (
    <div className="flex items-center justify-between mb-6">
      <h1 className="text-xl font-bold tracking-tight text-slate-900">{title}</h1>
      {action}
    </div>
  );
}

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  return (
    <div
      className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm flex items-center justify-center z-50 p-4 animate-fade"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-2xl shadow-2xl shadow-slate-900/20 ring-1 ring-slate-900/5 w-full max-w-lg max-h-[90vh] overflow-auto animate-pop"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-100 sticky top-0 bg-white/90 backdrop-blur">
          <h2 className="font-semibold text-slate-900">{title}</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700 w-8 h-8 grid place-items-center rounded-lg hover:bg-slate-100 transition">✕</button>
        </div>
        <div className="p-5">{children}</div>
      </div>
    </div>
  );
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block mb-3">
      <span className="block text-xs font-medium text-slate-500 mb-1.5">{label}</span>
      {children}
    </label>
  );
}

export const inputClass =
  "w-full bg-white rounded-lg ring-1 ring-slate-200 px-3 py-2 text-sm text-slate-800 placeholder:text-slate-400 transition focus:outline-none focus:ring-2 focus:ring-brand-500";

export function Btn({
  children,
  onClick,
  variant = "primary",
  type = "button",
  disabled,
}: {
  children: ReactNode;
  onClick?: () => void;
  variant?: "primary" | "ghost" | "danger";
  type?: "button" | "submit";
  disabled?: boolean;
}) {
  const styles = {
    primary: "bg-brand-600 text-white hover:bg-brand-500 shadow-sm shadow-brand-600/20",
    ghost: "bg-slate-100 text-slate-600 hover:bg-slate-200",
    danger: "bg-rose-50 text-rose-600 hover:bg-rose-100",
  }[variant];
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      className={`inline-flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium transition active:scale-[0.97] disabled:opacity-50 disabled:pointer-events-none cursor-pointer ${styles}`}
    >
      {children}
    </button>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    pending: "bg-slate-100 text-slate-600",
    paid: "bg-emerald-100 text-emerald-700",
    processing: "bg-blue-100 text-blue-700",
    shipped: "bg-brand-100 text-brand-700",
    completed: "bg-emerald-100 text-emerald-700",
    cancelled: "bg-rose-100 text-rose-700",
  };
  return <span className={`badge ${colors[status] ?? "bg-slate-100 text-slate-600"}`}>{status}</span>;
}
