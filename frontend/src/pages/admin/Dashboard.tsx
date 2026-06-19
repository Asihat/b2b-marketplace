import { useEffect, useState } from "react";
import { adminApi, type Dashboard as DashboardData } from "../../api";
import { PageHeader, StatusBadge } from "../../components/ui";

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="card card-hover p-4">
      <div className="text-2xl font-bold text-slate-900">{value}</div>
      <div className="text-xs text-slate-400 mt-1">{label}</div>
    </div>
  );
}

export function Dashboard() {
  const [data, setData] = useState<DashboardData | null>(null);

  useEffect(() => {
    adminApi.dashboard().then(setData).catch(() => {});
  }, []);

  if (!data) return <p className="text-slate-400">Loading dashboard…</p>;
  const { stats } = data;

  return (
    <div>
      <PageHeader title="Dashboard" />

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-6">
        <Stat label="Users" value={stats.users} />
        <Stat label="B2B users" value={stats.b2b_users} />
        <Stat label="Companies" value={stats.companies} />
        <Stat label="Products (active)" value={`${stats.active_products}/${stats.products}`} />
        <Stat label="Orders" value={stats.orders} />
        <Stat label="Pending orders" value={stats.pending_orders} />
        <Stat
          label="Revenue"
          value={
            Object.entries(stats.revenue_by_currency)
              .map(([c, v]) => `${Number(v).toFixed(0)} ${c}`)
              .join(" · ") || "—"
          }
        />
      </div>

      <div className="grid lg:grid-cols-2 gap-4">
        <div className="card p-4">
          <h2 className="font-semibold text-slate-700 mb-3">Recent orders</h2>
          <table className="w-full text-sm">
            <tbody>
              {data.recent_orders.map((o) => (
                <tr key={o.id} className="border-t border-slate-100">
                  <td className="py-2 font-medium">{o.number}</td>
                  <td className="text-slate-500">{o.user?.name ?? "—"}</td>
                  <td className="text-right">{Number(o.grand_total).toFixed(2)} {o.currency_code}</td>
                  <td className="text-right pl-2"><StatusBadge status={o.status} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <div className="card p-4">
          <h2 className="font-semibold text-slate-700 mb-3">Lowest stock</h2>
          <table className="w-full text-sm">
            <tbody>
              {data.low_stock.map((p) => (
                <tr key={p.id} className="border-t border-slate-100">
                  <td className="py-2 font-medium">{p.name}</td>
                  <td className="text-slate-400">{p.sku}</td>
                  <td className={`text-right ${p.stock <= p.min_order_qty ? "text-red-600 font-semibold" : ""}`}>
                    {p.stock}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
