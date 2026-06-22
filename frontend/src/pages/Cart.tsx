import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useStore } from "../store";
import { useT } from "../i18n";
import { formatMoney } from "../lib/format";

export function Cart() {
  const { cart, currency, removeFromCart, clearCart, user } = useStore();
  const t = useT();
  const navigate = useNavigate();
  const [gateways, setGateways] = useState<string[]>([]);
  const [gateway, setGateway] = useState("fake");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const [form, setForm] = useState({
    contact_name: "",
    contact_email: "",
    contact_phone: "",
    shipping_address: "",
    shipping_city: "",
    shipping_postal_code: "",
    shipping_country: "",
    notes: "",
  });

  useEffect(() => {
    api.gateways().then((g) => { setGateways(g.available); setGateway(g.default); }).catch(() => {});
  }, []);

  // Prefill contact details from the signed-in user.
  useEffect(() => {
    if (user) {
      setForm((f) => ({
        ...f,
        contact_name: f.contact_name || user.name,
        contact_email: f.contact_email || user.email,
      }));
    }
  }, [user]);

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
    setForm({ ...form, [k]: e.target.value });

  const total = cart.reduce((sum, l) => sum + l.product.price.amount * l.quantity, 0);

  async function checkout() {
    setError(""); setMessage("");
    if (!user) { navigate("/login"); return; }
    if (!form.shipping_address.trim()) { setError(t("cart.addressRequired")); return; }
    setBusy(true);
    try {
      const order = await api.createOrder({
        currency_code: currency,
        items: cart.map((l) => ({ product_id: l.product.id, quantity: l.quantity })),
        ...form,
      });
      const paid = await api.pay(order.id, gateway);
      setMessage(t("cart.orderPlaced", {
        number: paid.order.number,
        status: t(`status.${paid.order.status}`),
        pstatus: paid.payment.status,
        gateway: paid.payment.gateway,
      }));
      clearCart();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Checkout failed");
    } finally {
      setBusy(false);
    }
  }

  if (cart.length === 0) {
    return (
      <div className="card py-20 text-center max-w-lg mx-auto">
        <div className="text-5xl mb-4">{message ? "✅" : "🛍️"}</div>
        <p className="text-slate-500 mb-4">{message || t("cart.empty")}</p>
        {message
          ? <Link to="/orders" className="btn-primary">{t("cart.viewOrders")}</Link>
          : <Link to="/" className="btn-soft">{t("cart.browse")}</Link>}
      </div>
    );
  }

  const input = "input";
  const label = "block text-xs font-medium text-slate-500 mb-1.5";

  return (
    <div className="grid lg:grid-cols-3 gap-6 items-start">
      <div className="lg:col-span-2 space-y-6">
        <div>
          <h1 className="text-xl font-bold text-slate-900 mb-4">{t("cart.title")} <span className="text-slate-400 font-normal">({cart.length})</span></h1>
          <div className="card divide-y divide-slate-100">
            {cart.map((l) => (
              <div key={l.product.id} className="flex items-center gap-4 p-4">
                <div className="w-16 h-16 rounded-xl overflow-hidden bg-slate-100 shrink-0">
                  {l.product.image && <img src={l.product.image} alt="" className="w-full h-full object-cover" />}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-slate-800 truncate">{l.product.name}</div>
                  <div className="text-xs text-slate-400">{l.product.sku} · {l.product.price.formatted} each</div>
                </div>
                <div className="text-sm text-slate-500">× {l.quantity}</div>
                <div className="w-28 text-right font-semibold text-slate-900">
                  {formatMoney(l.product.price.amount * l.quantity, currency)}
                </div>
                <button onClick={() => removeFromCart(l.product.id)} className="text-slate-300 hover:text-rose-600 transition w-7 h-7 grid place-items-center rounded-lg">✕</button>
              </div>
            ))}
          </div>
        </div>

        {/* Contact & delivery details */}
        <div className="card p-5">
          <h2 className="font-semibold text-slate-900 mb-4">{t("cart.contactDelivery")}</h2>
          <div className="grid sm:grid-cols-2 gap-4">
            <div>
              <label className={label}>{t("cart.contactName")}</label>
              <input className={input} value={form.contact_name} onChange={set("contact_name")} />
            </div>
            <div>
              <label className={label}>{t("cart.emailUpdates")}</label>
              <input type="email" className={input} value={form.contact_email} onChange={set("contact_email")} />
            </div>
            <div>
              <label className={label}>{t("cart.phone")}</label>
              <input className={input} value={form.contact_phone} onChange={set("contact_phone")} />
            </div>
            <div>
              <label className={label}>{t("cart.country")}</label>
              <input maxLength={2} className={input} value={form.shipping_country} onChange={(e) => setForm({ ...form, shipping_country: e.target.value.toUpperCase() })} />
            </div>
            <div className="sm:col-span-2">
              <label className={label}>{t("cart.shippingAddress")} <span className="text-rose-500">*</span></label>
              <textarea rows={2} className={input} value={form.shipping_address} onChange={set("shipping_address")} />
            </div>
            <div>
              <label className={label}>{t("cart.city")}</label>
              <input className={input} value={form.shipping_city} onChange={set("shipping_city")} />
            </div>
            <div>
              <label className={label}>{t("cart.postal")}</label>
              <input className={input} value={form.shipping_postal_code} onChange={set("shipping_postal_code")} />
            </div>
            <div className="sm:col-span-2">
              <label className={label}>{t("cart.notes")}</label>
              <textarea rows={2} className={input} value={form.notes} onChange={set("notes")} />
            </div>
          </div>
        </div>
      </div>

      <div className="card p-5 lg:sticky lg:top-24">
        <h2 className="font-semibold text-slate-900 mb-4">{t("cart.summary")}</h2>
        <div className="flex justify-between text-sm text-slate-500 mb-2">
          <span>{t("cart.items")}</span><span>{cart.reduce((n, l) => n + l.quantity, 0)}</span>
        </div>
        <div className="flex justify-between text-lg font-semibold text-slate-900 pt-3 border-t border-slate-100">
          <span>{t("common.total")}</span><span>{formatMoney(total, currency)}</span>
        </div>

        <label className="block text-xs font-medium text-slate-500 mt-5 mb-1.5">{t("cart.payment")}</label>
        <select value={gateway} onChange={(e) => setGateway(e.target.value)} className="input capitalize">
          {gateways.map((g) => <option key={g} value={g}>{g}</option>)}
        </select>

        <button onClick={checkout} disabled={busy} className="btn-primary w-full mt-4 !py-2.5">
          {busy ? t("cart.processing") : user ? t("cart.checkout") : t("cart.signinToCheckout")}
        </button>
        {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mt-3">{error}</p>}
      </div>
    </div>
  );
}
