/** Order lifecycle statuses (mirrors the backend OrderStatus enum). */
export const ORDER_STATUSES = [
  "pending",
  "paid",
  "processing",
  "shipped",
  "completed",
  "cancelled",
] as const;

export type OrderStatus = (typeof ORDER_STATUSES)[number];

/** Tailwind badge classes per order status, shared by storefront + admin. */
export const STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-100 text-slate-600",
  paid: "bg-emerald-100 text-emerald-700",
  processing: "bg-blue-100 text-blue-700",
  shipped: "bg-brand-100 text-brand-700",
  completed: "bg-emerald-100 text-emerald-700",
  cancelled: "bg-rose-100 text-rose-700",
};
