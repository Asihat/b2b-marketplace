import { useEffect, useState } from "react";
import { adminApi, type Order } from "../../api";
import { PageHeader, StatusBadge, Modal, Btn, inputClass } from "../../components/ui";
import { ORDER_STATUSES } from "../../lib/constants";
import { formatMoney } from "../../lib/format";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";

type AdminOrder = Order & { user?: { name: string; email: string } };

export function Orders() {
  const [orders, setOrders] = useState<AdminOrder[]>([]);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [viewing, setViewing] = useState<AdminOrder | null>(null);
  const debouncedSearch = useDebouncedValue(search);

  useEffect(() => {
    adminApi.orders({ status, search: debouncedSearch }).then((r) => setOrders(r.data)).catch(() => {});
  }, [status, debouncedSearch]);

  async function changeStatus(id: number, value: string) {
    await adminApi.setOrderStatus(id, value);
    setOrders((prev) => prev.map((o) => (o.id === id ? { ...o, status: value } : o)));
  }

  async function view(id: number) {
    const full = await adminApi.order(id);
    setViewing(full);
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
          {ORDER_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
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
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {orders.map((o) => (
              <tr key={o.id} className="border-t border-slate-100">
                <td className="px-4 py-2 font-medium">{o.number}</td>
                <td className="px-4 py-2 text-slate-500">{o.user?.name ?? "—"}</td>
                <td className="px-4 py-2 uppercase text-xs">{o.type}</td>
                <td className="px-4 py-2 text-right">{formatMoney(o.grand_total, o.currency_code)}</td>
                <td className="px-4 py-2"><StatusBadge status={o.status} /></td>
                <td className="px-4 py-2">
                  <select
                    value={o.status}
                    onChange={(e) => changeStatus(o.id, e.target.value)}
                    className="border border-slate-300 rounded-md px-2 py-1 text-xs"
                  >
                    {ORDER_STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
                  </select>
                </td>
                <td className="px-4 py-2 text-right">
                  <Btn variant="ghost" onClick={() => view(o.id)}>View</Btn>
                </td>
              </tr>
            ))}
            {orders.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-slate-400">No orders.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {viewing && (
        <Modal title={`Order ${viewing.number}`} onClose={() => setViewing(null)}>
          <div className="flex items-center gap-2 mb-4">
            <StatusBadge status={viewing.status} />
            <span className="text-xs uppercase text-slate-400">{viewing.type}</span>
          </div>

          <Section title="Contact">
            <Row label="Name" value={viewing.contact_name} />
            <Row label="Email" value={viewing.contact_email} />
            <Row label="Phone" value={viewing.contact_phone} />
          </Section>

          <Section title="Shipping">
            <Row label="Address" value={viewing.shipping_address} />
            <Row label="City" value={viewing.shipping_city} />
            <Row label="Postal" value={viewing.shipping_postal_code} />
            <Row label="Country" value={viewing.shipping_country} />
          </Section>

          {viewing.notes && (
            <Section title="Notes">
              <p className="text-sm text-slate-600">{viewing.notes}</p>
            </Section>
          )}

          <Section title="Items">
            <div className="divide-y divide-slate-100">
              {viewing.items?.map((it) => (
                <div key={it.id} className="flex justify-between py-2 text-sm">
                  <span className="text-slate-700">{it.name} <span className="text-slate-400">× {it.quantity}</span></span>
                  <span className="font-medium">{formatMoney(it.line_total, viewing.currency_code)}</span>
                </div>
              ))}
            </div>
            <div className="flex justify-between pt-3 mt-2 border-t border-slate-200 font-semibold">
              <span>Total</span>
              <span>{formatMoney(viewing.grand_total, viewing.currency_code)}</span>
            </div>
          </Section>
        </Modal>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="mb-4">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-1.5">{title}</h3>
      {children}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string | null }) {
  return (
    <div className="flex justify-between text-sm py-0.5">
      <span className="text-slate-400">{label}</span>
      <span className="text-slate-700 text-right">{value || "—"}</span>
    </div>
  );
}
