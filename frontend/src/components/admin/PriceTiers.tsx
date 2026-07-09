import { useMemo, useState } from "react";
import { adminApi, type Currency, type ProductPriceRow } from "../../api";
import { Btn, inputClass } from "../ui";

/**
 * The full price grid of a product, edited in place and saved in one request.
 *
 * A row is a per-currency price that applies from `min_qty` upwards. Whichever
 * row has the highest `min_qty` at or below the ordered quantity wins; with no
 * matching row the product falls back to its base price. This mirrors
 * CurrencyService::priceFor() on the server.
 */

interface Row {
  key: number;
  currency_code: string;
  min_qty: string;
  price: string;
}

let nextKey = 1;

function toRows(prices: ProductPriceRow[]): Row[] {
  return [...prices]
    .sort((a, b) => a.currency_code.localeCompare(b.currency_code) || a.min_qty - b.min_qty)
    .map((p) => ({
      key: nextKey++,
      currency_code: p.currency_code,
      min_qty: String(p.min_qty),
      price: Number(p.price).toString(),
    }));
}

/** Stable representation used to detect unsaved edits. */
function fingerprint(rows: Row[]): string {
  return JSON.stringify(rows.map((r) => [r.currency_code, r.min_qty, r.price]));
}

/**
 * Resolve the price a buyer would actually pay — the same order of preference
 * as CurrencyService::priceFor(). An override entered for the requested
 * currency is used verbatim; base-currency tiers and the base price are
 * converted at `rate`.
 */
function resolvePrice(rows: Row[], basePrice: number, currency: string, baseCurrency: string, qty: number, rate: number) {
  const bestFor = (code: string) =>
    rows
      .filter((r) => r.currency_code === code && Number(r.min_qty) <= qty && r.price !== "")
      .sort((a, b) => Number(b.min_qty) - Number(a.min_qty))[0];

  const own = bestFor(currency);
  if (own) return { amount: Number(own.price), source: `tier from ${own.min_qty}+` };

  if (currency !== baseCurrency) {
    const baseTier = bestFor(baseCurrency);
    if (baseTier) {
      return {
        amount: Number(baseTier.price) * rate,
        source: `${baseCurrency} tier from ${baseTier.min_qty}+, converted`,
      };
    }
  }

  return {
    amount: basePrice * rate,
    source: currency === baseCurrency ? "base price" : "base price, converted",
  };
}

export function PriceTiers({
  productId,
  basePrice,
  currencies,
  initial,
  onSaved,
}: {
  productId: number;
  basePrice: number;
  currencies: Currency[];
  initial: ProductPriceRow[];
  onSaved?: (rows: ProductPriceRow[]) => void;
}) {
  const [saved, setSaved] = useState<Row[]>(() => toRows(initial));
  const [rows, setRows] = useState<Row[]>(saved);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const [previewCurrency, setPreviewCurrency] = useState("");
  const [previewQty, setPreviewQty] = useState(1);

  const baseCurrency = currencies.find((c) => c.is_base)?.code ?? currencies[0]?.code ?? "USD";
  const dirty = fingerprint(rows) !== fingerprint(saved);

  /** Units of `code` per unit of the base currency. */
  const rateOf = (code: string) => Number(currencies.find((c) => c.code === code)?.exchange_rate ?? 1) || 1;

  /** Rows that collide on (currency, min qty) — the server rejects these too. */
  const duplicates = useMemo(() => {
    const seen = new Map<string, number>();
    const clashing = new Set<number>();
    for (const row of rows) {
      const tier = `${row.currency_code}|${row.min_qty}`;
      const first = seen.get(tier);
      if (first !== undefined) {
        clashing.add(first);
        clashing.add(row.key);
      } else {
        seen.set(tier, row.key);
      }
    }
    return clashing;
  }, [rows]);

  const invalid = rows.some(
    (r) => !r.currency_code || !(Number(r.min_qty) >= 1) || r.price === "" || !(Number(r.price) >= 0),
  );

  const patch = (key: number, changes: Partial<Row>) =>
    setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...changes } : r)));

  function addRow() {
    const currency = rows.at(-1)?.currency_code ?? baseCurrency;
    const highestTier = Math.max(0, ...rows.filter((r) => r.currency_code === currency).map((r) => Number(r.min_qty)));
    setRows((rs) => [
      ...rs,
      {
        key: nextKey++,
        currency_code: currency,
        min_qty: String(highestTier ? highestTier * 10 : 1),
        price: basePrice ? (basePrice * rateOf(currency)).toFixed(2) : "",
      },
    ]);
  }

  function reset() {
    setRows(saved);
    setError("");
  }

  async function save() {
    setSaving(true);
    setError("");
    try {
      const persisted = await adminApi.syncProductPrices(
        productId,
        rows.map((r) => ({
          currency_code: r.currency_code,
          min_qty: Number(r.min_qty),
          price: Number(r.price),
        })),
      );
      const next = toRows(persisted);
      setRows(next);
      setSaved(next);
      onSaved?.(persisted);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Prices could not be saved.");
    } finally {
      setSaving(false);
    }
  }

  const previewIn = previewCurrency || baseCurrency;
  const preview = resolvePrice(rows, basePrice, previewIn, baseCurrency, Math.max(1, previewQty), rateOf(previewIn));

  return (
    <div className="card p-5">
      <div className="flex items-start justify-between gap-4 mb-4">
        <div>
          <h2 className="text-base font-semibold text-slate-900">Pricing</h2>
          <p className="text-sm text-slate-500 mt-0.5">
            Each row sets the unit price from a minimum quantity upwards, in that currency. Without a matching row the
            base price of <span className="font-medium text-slate-700">{basePrice.toFixed(2)} {baseCurrency}</span> is
            converted at the currency's exchange rate.
          </p>
        </div>
        {dirty && (
          <span className="badge bg-amber-100 text-amber-700 shrink-0">Unsaved</span>
        )}
      </div>

      <table className="w-full text-sm">
        <thead className="text-left text-xs text-slate-400">
          <tr>
            <th className="pb-2 font-medium">Currency</th>
            <th className="pb-2 font-medium">From qty</th>
            <th className="pb-2 font-medium">Unit price</th>
            <th className="pb-2 font-medium text-right">vs base</th>
            <th className="pb-2 w-10"></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const clash = duplicates.has(row.key);
            // Compare against the base price expressed in this row's own currency.
            const rowBase = basePrice * rateOf(row.currency_code);
            const delta = rowBase > 0 && row.price !== "" ? (Number(row.price) - rowBase) / rowBase : null;
            return (
              <tr key={row.key} className="border-t border-slate-100">
                <td className="py-2 pr-2">
                  <select
                    className={inputClass + " !py-1.5"}
                    value={row.currency_code}
                    onChange={(e) => patch(row.key, { currency_code: e.target.value })}
                  >
                    {currencies.map((c) => (
                      <option key={c.code} value={c.code}>{c.code}</option>
                    ))}
                  </select>
                </td>
                <td className="py-2 pr-2">
                  <input
                    type="number"
                    min={1}
                    value={row.min_qty}
                    onChange={(e) => patch(row.key, { min_qty: e.target.value })}
                    className={`${inputClass} !py-1.5 w-24 ${clash ? "!ring-rose-400 !ring-2" : ""}`}
                  />
                </td>
                <td className="py-2 pr-2">
                  <input
                    type="number"
                    step="0.0001"
                    min={0}
                    value={row.price}
                    onChange={(e) => patch(row.key, { price: e.target.value })}
                    className={inputClass + " !py-1.5 w-32"}
                  />
                </td>
                <td className="py-2 pr-2 text-right tabular-nums">
                  {delta === null || delta === 0 ? (
                    <span className="text-slate-300">—</span>
                  ) : (
                    <span className={delta < 0 ? "text-emerald-600" : "text-slate-500"}>
                      {delta > 0 ? "+" : ""}{(delta * 100).toFixed(0)}%
                    </span>
                  )}
                </td>
                <td className="py-2 text-right">
                  <Btn variant="danger" onClick={() => setRows((rs) => rs.filter((r) => r.key !== row.key))}>✕</Btn>
                </td>
              </tr>
            );
          })}
          {rows.length === 0 && (
            <tr>
              <td colSpan={5} className="py-6 text-center text-slate-400">
                No overrides — the base price applies in every currency.
              </td>
            </tr>
          )}
        </tbody>
      </table>

      {duplicates.size > 0 && (
        <p className="text-sm text-rose-600 mt-3">Two rows share the same currency and minimum quantity.</p>
      )}
      {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mt-3">{error}</p>}

      <div className="flex flex-wrap items-center justify-between gap-2 mt-4">
        <Btn variant="ghost" onClick={addRow}>+ Add tier</Btn>
        <div className="flex gap-2">
          {dirty && <Btn variant="ghost" onClick={reset}>Discard</Btn>}
          <Btn onClick={save} disabled={!dirty || saving || invalid || duplicates.size > 0}>
            {saving ? "Saving..." : "Save prices"}
          </Btn>
        </div>
      </div>

      <div className="mt-5 pt-4 border-t border-slate-100">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-400 mb-2">What a buyer pays</h3>
        <div className="flex flex-wrap items-end gap-2">
          <div className="w-28">
            <label className="block text-xs text-slate-500 mb-1">Currency</label>
            <select
              className={inputClass + " !py-1.5"}
              value={previewCurrency || baseCurrency}
              onChange={(e) => setPreviewCurrency(e.target.value)}
            >
              {currencies.map((c) => <option key={c.code} value={c.code}>{c.code}</option>)}
            </select>
          </div>
          <div className="w-24">
            <label className="block text-xs text-slate-500 mb-1">Quantity</label>
            <input
              type="number"
              min={1}
              className={inputClass + " !py-1.5"}
              value={previewQty}
              onChange={(e) => setPreviewQty(Number(e.target.value))}
            />
          </div>
          <div className="flex-1 min-w-48 rounded-lg bg-slate-50 ring-1 ring-slate-200 px-3 py-2">
            <span className="text-lg font-semibold text-slate-900 tabular-nums">{preview.amount.toFixed(2)}</span>
            <span className="text-sm text-slate-500"> {previewCurrency || baseCurrency} · {preview.source}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
