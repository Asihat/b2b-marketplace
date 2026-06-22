import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type Order } from "../api";
import { useStore } from "../store";
import { useT } from "../i18n";
import { STATUS_COLORS } from "../lib/constants";
import { formatMoney } from "../lib/format";

export function Orders() {
  const { user } = useStore();
  const t = useT();
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
        <Link to="/login" className="text-brand-600 font-medium hover:underline">{t("orders.signin")}</Link> {t("orders.signinToView")}
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
        <div className="text-5xl mb-4">📦</div>{t("orders.none")}
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-xl font-bold text-slate-900 mb-4">{t("orders.title")}</h1>
      <div className="space-y-3">
        {orders.map((o) => (
          <div key={o.id} className="card p-5">
            <div className="flex items-center justify-between gap-3">
              <span className="font-semibold text-slate-900">{o.number}</span>
              <span className={`badge ${STATUS_COLORS[o.status] ?? "bg-slate-100 text-slate-600"}`}>{t(`status.${o.status}`)}</span>
            </div>
            <div className="text-sm text-slate-500 mt-1">
              <span className="uppercase text-xs tracking-wide">{o.type}</span> · {formatMoney(o.grand_total, o.currency_code)}
              {o.payment && ` · ${t("orders.paidVia", { gateway: o.payment.gateway })}`}
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
            {(o.shipping_address || o.contact_email) && (
              <div className="mt-3 pt-3 border-t border-slate-100 text-xs text-slate-500 space-y-0.5">
                {o.contact_name && <div>{o.contact_name}{o.contact_phone ? ` · ${o.contact_phone}` : ""}</div>}
                {o.contact_email && <div>{o.contact_email}</div>}
                {o.shipping_address && (
                  <div>
                    📍 {[o.shipping_address, o.shipping_city, o.shipping_postal_code, o.shipping_country].filter(Boolean).join(", ")}
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
