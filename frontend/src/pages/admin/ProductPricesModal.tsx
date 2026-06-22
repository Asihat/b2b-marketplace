import { useEffect, useState } from "react";
import { adminApi, api, type AdminProduct, type Currency, type ProductPriceRow } from "../../api";
import { Modal, Btn, inputClass } from "../../components/ui";

/**
 * Manage a product's per-currency price overrides and B2B volume tiers.
 * When no override matches, the storefront falls back to the product's base price.
 */
export function ProductPricesModal({ product, onClose }: { product: AdminProduct; onClose: () => void }) {
  const [rows, setRows] = useState<ProductPriceRow[]>([]);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [draft, setDraft] = useState({ currency_code: "", min_qty: 1, price: "" });
  const [error, setError] = useState("");

  function load() {
    adminApi.productPrices(product.id).then(setRows).catch(() => {});
  }

  useEffect(() => {
    load();
    api.currencies().then((c) => {
      setCurrencies(c);
      setDraft((d) => ({ ...d, currency_code: d.currency_code || c[0]?.code || "" }));
    }).catch(() => {});
  }, []);

  async function add() {
    setError("");
    try {
      await adminApi.saveProductPrice(product.id, null, {
        currency_code: draft.currency_code,
        min_qty: Number(draft.min_qty),
        price: Number(draft.price),
      });
      setDraft({ ...draft, min_qty: 1, price: "" });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to add price");
    }
  }

  async function save(row: ProductPriceRow) {
    setError("");
    try {
      await adminApi.saveProductPrice(product.id, row.id, {
        currency_code: row.currency_code,
        min_qty: row.min_qty,
        price: Number(row.price),
      });
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save");
    }
  }

  async function remove(id: number) {
    await adminApi.deleteProductPrice(product.id, id);
    load();
  }

  const setPrice = (id: number, price: string) =>
    setRows((rs) => rs.map((r) => (r.id === id ? { ...r, price } : r)));

  return (
    <Modal title={`Prices — ${product.name}`} onClose={onClose}>
      <p className="text-xs text-slate-500 mb-4">
        Base price: <span className="font-semibold text-slate-700">{Number(product.base_price).toFixed(2)}</span>{" "}
        (used when no override below matches). Add a row to set a fixed price for a currency, or a cheaper
        per-unit price from a minimum quantity (B2B volume tier).
      </p>

      <table className="w-full text-sm mb-4">
        <thead className="text-slate-400 text-left text-xs">
          <tr>
            <th className="py-1">Currency</th>
            <th className="py-1">Min qty</th>
            <th className="py-1">Price</th>
            <th className="py-1"></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="border-t border-slate-100">
              <td className="py-2 font-mono">{row.currency_code}</td>
              <td className="py-2">{row.min_qty}</td>
              <td className="py-2">
                <input
                  type="number"
                  step="0.0001"
                  value={row.price}
                  onChange={(e) => setPrice(row.id, e.target.value)}
                  className={inputClass + " !py-1 w-28"}
                />
              </td>
              <td className="py-2 text-right space-x-1 whitespace-nowrap">
                <Btn variant="ghost" onClick={() => save(row)}>Save</Btn>
                <Btn variant="danger" onClick={() => remove(row.id)}>✕</Btn>
              </td>
            </tr>
          ))}
          {rows.length === 0 && (
            <tr><td colSpan={4} className="py-3 text-center text-slate-400">No overrides — base price applies everywhere.</td></tr>
          )}
        </tbody>
      </table>

      <div className="border-t border-slate-200 pt-4">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">Add price tier</h3>
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <label className="block text-xs text-slate-500 mb-1">Currency</label>
            <select
              className={inputClass + " !py-2"}
              value={draft.currency_code}
              onChange={(e) => setDraft({ ...draft, currency_code: e.target.value })}
            >
              {currencies.map((c) => <option key={c.code} value={c.code}>{c.code}</option>)}
            </select>
          </div>
          <div className="w-24">
            <label className="block text-xs text-slate-500 mb-1">Min qty</label>
            <input
              type="number"
              min={1}
              className={inputClass + " !py-2"}
              value={draft.min_qty}
              onChange={(e) => setDraft({ ...draft, min_qty: Number(e.target.value) })}
            />
          </div>
          <div className="w-28">
            <label className="block text-xs text-slate-500 mb-1">Price</label>
            <input
              type="number"
              step="0.0001"
              className={inputClass + " !py-2"}
              value={draft.price}
              onChange={(e) => setDraft({ ...draft, price: e.target.value })}
            />
          </div>
          <Btn onClick={add}>Add</Btn>
        </div>
      </div>

      {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mt-3">{error}</p>}
    </Modal>
  );
}
