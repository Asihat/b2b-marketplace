import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type Order } from "../api";
import { useStore } from "../store";

const STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-100 text-slate-600",
  paid: "bg-emerald-100 text-emerald-700",
  processing: "bg-blue-100 text-blue-700",
  shipped: "bg-brand-100 text-brand-700",
  completed: "bg-emerald-100 text-emerald-700",
  cancelled: "bg-rose-100 text-rose-700",
};

export function Orders() {
  const { user } = useStore();
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) return;
    api.orders().then((res) => setOrders(res.data)).catch(() => {}).finally(() => setLoading(false));
  }, [user]);

  if (!user) {
    return (
      <div className="card py-20 text-center max-w-lg mx-auto text-slate-500">
        <div className="text-5xl mb-4">🔒</div>
        <Link to="/login" className="text-brand-600 font-medium hover:underline">Sign in</Link> to view your orders.
      </div>
    );
  }

  if (loading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => <div key={i} className="card h-24 skeleton" />)}
      </div>
    );
  }

  if (orders.length === 0) {
    return (
      <div className="card py-20 text-center max-w-lg mx-auto text-slate-500">
        <div className="text-5xl mb-4">📦</div>No orders yet.
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-xl font-bold text-slate-900 mb-4">Your orders</h1>
      <div className="space-y-3">
        {orders.map((o) => (
          <div key={o.id} className="card p-5">
            <div className="flex items-center justify-between gap-3">
              <span className="font-semibold text-slate-900">{o.number}</span>
              <span className={`badge ${STATUS_COLORS[o.status] ?? "bg-slate-100 text-slate-600"}`}>{o.status}</span>
            </div>
            <div className="text-sm text-slate-500 mt-1">
              <span className="uppercase text-xs tracking-wide">{o.type}</span> · {Number(o.grand_total).toFixed(2)} {o.currency_code}
              {o.payment && ` · paid via ${o.payment.gateway}`}
            </div>
            {o.items && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                {o.items.map((it) => (
                  <span key={it.id} className="badge bg-slate-50 ring-1 ring-slate-200 text-slate-600">
                    {it.name} × {it.quantity}
                  </span>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
