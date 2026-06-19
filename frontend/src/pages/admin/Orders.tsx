import { useEffect, useState } from "react";
import { adminApi, type Order } from "../../api";
import { PageHeader, StatusBadge, inputClass } from "../../components/ui";

const STATUSES = ["pending", "paid", "processing", "shipped", "completed", "cancelled"];

type AdminOrder = Order & { user?: { name: string; email: string } };

export function Orders() {
  const [orders, setOrders] = useState<AdminOrder[]>([]);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");

  function load() {
    adminApi.orders({ status, search }).then((r) => setOrders(r.data)).catch(() => {});
  }

  useEffect(() => {
    const t = setTimeout(load, 200);
    return () => clearTimeout(t);
  }, [status, search]);

  async function changeStatus(id: number, value: string) {
    await adminApi.setOrderStatus(id, value);
    setOrders((prev) => prev.map((o) => (o.id === id ? { ...o, status: value } : o)));
  }

  return (
    <div>
      <PageHeader title="Orders" />

      <div className="flex gap-3 mb-4">
        <input
          placeholder="Search order #…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className={inputClass + " max-w-xs"}
        />
        <select value={status} onChange={(e) => setStatus(e.target.value)} className={inputClass + " max-w-[180px]"}>
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </div>

      <div className="card overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2">Order</th>
              <th className="px-4 py-2">Customer</th>
              <th className="px-4 py-2">Type</th>
              <th className="px-4 py-2 text-right">Total</th>
              <th className="px-4 py-2">Status</th>
              <th className="px-4 py-2">Set status</th>
            </tr>
          </thead>
          <tbody>
            {orders.map((o) => (
              <tr key={o.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-medium">{o.number}</td>
                <td className="px-4 py-2 text-slate-500">{o.user?.name ?? "—"}</td>
                <td className="px-4 py-2 uppercase text-xs">{o.type}</td>
                <td className="px-4 py-2 text-right">{Number(o.grand_total).toFixed(2)} {o.currency_code}</td>
                <td className="px-4 py-2"><StatusBadge status={o.status} /></td>
                <td className="px-4 py-2">
                  <select
                    value={o.status}
                    onChange={(e) => changeStatus(o.id, e.target.value)}
                    className="border border-slate-300 rounded-md px-2 py-1 text-xs"
                  >
                    {STATUSES.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
            {orders.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-slate-400">No orders.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
