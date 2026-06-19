import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useStore } from "../store";

export function Cart() {
  const { cart, currency, removeFromCart, clearCart, user } = useStore();
  const navigate = useNavigate();
  const [gateways, setGateways] = useState<string[]>([]);
  const [gateway, setGateway] = useState("fake");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    api.gateways().then((g) => { setGateways(g.available); setGateway(g.default); }).catch(() => {});
  }, []);

  const total = cart.reduce((sum, l) => sum + l.product.price.amount * l.quantity, 0);

  async function checkout() {
    setError(""); setMessage("");
    if (!user) { navigate("/login"); return; }
    setBusy(true);
    try {
      const order = await api.createOrder({
        currency_code: currency,
        items: cart.map((l) => ({ product_id: l.product.id, quantity: l.quantity })),
      });
      const paid = await api.pay(order.id, gateway);
      setMessage(`Order ${paid.order.number} placed — ${paid.order.status} (payment ${paid.payment.status} via ${paid.payment.gateway}).`);
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
        <p className="text-slate-500 mb-4">{message || "Your cart is empty."}</p>
        {message
          ? <Link to="/orders" className="btn-primary">View your orders</Link>
          : <Link to="/" className="btn-soft">Browse the catalog</Link>}
      </div>
    );
  }

  return (
    <div className="grid lg:grid-cols-3 gap-6 items-start">
      <div className="lg:col-span-2">
        <h1 className="text-xl font-bold text-slate-900 mb-4">Cart <span className="text-slate-400 font-normal">({cart.length})</span></h1>
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
                {(l.product.price.amount * l.quantity).toFixed(2)} {currency}
              </div>
              <button onClick={() => removeFromCart(l.product.id)} className="text-slate-300 hover:text-rose-600 transition w-7 h-7 grid place-items-center rounded-lg">✕</button>
            </div>
          ))}
        </div>
      </div>

      <div className="card p-5 lg:sticky lg:top-24">
        <h2 className="font-semibold text-slate-900 mb-4">Summary</h2>
        <div className="flex justify-between text-sm text-slate-500 mb-2">
          <span>Items</span><span>{cart.reduce((n, l) => n + l.quantity, 0)}</span>
        </div>
        <div className="flex justify-between text-lg font-semibold text-slate-900 pt-3 border-t border-slate-100">
          <span>Total</span><span>{total.toFixed(2)} {currency}</span>
        </div>

        <label className="block text-xs font-medium text-slate-500 mt-5 mb-1.5">Payment method</label>
        <select value={gateway} onChange={(e) => setGateway(e.target.value)} className="input capitalize">
          {gateways.map((g) => <option key={g} value={g}>{g}</option>)}
        </select>

        <button onClick={checkout} disabled={busy} className="btn-primary w-full mt-4 !py-2.5">
          {busy ? "Processing…" : user ? "Checkout & pay" : "Sign in to checkout"}
        </button>
        {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mt-3">{error}</p>}
      </div>
    </div>
  );
}
