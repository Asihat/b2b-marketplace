import { useEffect, useState } from "react";
import { adminApi, type AdminAnalog, type AdminProduct, type AnalogType } from "../../api";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { Btn, inputClass } from "../ui";

/**
 * Analog / cross-reference links for a product. Links are symmetric: adding B
 * as an analog of A also makes A an analog of B, so a link only has to be
 * managed from one side.
 */

const TYPES: { value: AnalogType; label: string; hint: string; badge: string }[] = [
  { value: "equivalent", label: "Equivalent", hint: "Interchangeable, same fit and function", badge: "bg-emerald-50 text-emerald-700" },
  { value: "substitute", label: "Substitute", hint: "Works instead of, with caveats", badge: "bg-amber-50 text-amber-700" },
  { value: "upgrade", label: "Upgrade", hint: "A better version of this product", badge: "bg-brand-50 text-brand-700" },
];

const badgeFor = (type: AnalogType) => TYPES.find((t) => t.value === type)?.badge ?? "bg-slate-100 text-slate-600";

function Thumb({ url }: { url: string | null }) {
  return (
    <div className="w-10 h-10 rounded-lg bg-slate-100 overflow-hidden shrink-0">
      {url
        ? <img src={url} alt="" className="w-full h-full object-cover" />
        : <div className="w-full h-full grid place-items-center text-slate-300">◆</div>}
    </div>
  );
}

export function AnalogLinks({ productId, onCountChange }: { productId: number; onCountChange?: (n: number) => void }) {
  const [analogs, setAnalogs] = useState<AdminAnalog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busyId, setBusyId] = useState<number | null>(null);

  const [query, setQuery] = useState("");
  const [results, setResults] = useState<AdminProduct[]>([]);
  const [searching, setSearching] = useState(false);
  const debouncedQuery = useDebouncedValue(query);

  function apply(next: AdminAnalog[]) {
    setAnalogs(next);
    onCountChange?.(next.length);
  }

  useEffect(() => {
    adminApi.productAnalogs(productId)
      .then(apply)
      .catch((e) => setError(e instanceof Error ? e.message : "Analogs could not be loaded."))
      .finally(() => setLoading(false));
  }, [productId]);

  useEffect(() => {
    const term = debouncedQuery.trim();
    if (term.length < 2) {
      setResults([]);
      return;
    }
    let stale = false;
    setSearching(true);
    adminApi.products({ search: term, per_page: 6, exclude_id: productId })
      .then((r) => { if (!stale) setResults(r.data); })
      .catch(() => { if (!stale) setResults([]); })
      .finally(() => { if (!stale) setSearching(false); });
    return () => { stale = true; };
  }, [debouncedQuery, productId]);

  async function run(id: number, action: () => Promise<AdminAnalog[]>) {
    setBusyId(id);
    setError("");
    try {
      apply(await action());
    } catch (e) {
      setError(e instanceof Error ? e.message : "The analog could not be updated.");
    } finally {
      setBusyId(null);
    }
  }

  const add = (product: AdminProduct) =>
    run(product.id, async () => {
      const next = await adminApi.addProductAnalog(productId, { analog_id: product.id, type: "equivalent" });
      setQuery("");
      return next;
    });

  const linkedIds = new Set(analogs.map((a) => a.id));
  const suggestions = results.filter((r) => !linkedIds.has(r.id));

  return (
    <div className="card p-5">
      <div className="mb-4">
        <h2 className="text-base font-semibold text-slate-900">Analogs</h2>
        <p className="text-sm text-slate-500 mt-0.5">
          Interchangeable products, shown to buyers on the product page. Links work both ways.
        </p>
      </div>

      <div className="relative mb-4">
        <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
        </svg>
        <input
          placeholder="Find a product by SKU, name or brand..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className={inputClass + " !pl-9"}
        />

        {query.trim().length >= 2 && (
          <div className="absolute z-10 mt-1 w-full card p-1 max-h-72 overflow-auto">
            {searching && <p className="px-3 py-2 text-sm text-slate-400">Searching...</p>}
            {!searching && suggestions.length === 0 && (
              <p className="px-3 py-2 text-sm text-slate-400">No other products match.</p>
            )}
            {!searching && suggestions.map((product) => (
              <button
                key={product.id}
                type="button"
                onClick={() => add(product)}
                disabled={busyId === product.id}
                className="w-full flex items-center gap-3 px-2 py-2 rounded-lg text-left hover:bg-slate-50 transition disabled:opacity-50"
              >
                <Thumb url={product.images?.[0]?.url ?? null} />
                <span className="min-w-0 flex-1">
                  <span className="block text-sm font-medium text-slate-800 truncate">{product.name}</span>
                  <span className="block text-xs text-slate-400 font-mono">{product.sku}</span>
                </span>
                <span className="text-sm text-brand-600 font-medium shrink-0">Link</span>
              </button>
            ))}
          </div>
        )}
      </div>

      {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mb-3">{error}</p>}

      {loading ? (
        <p className="py-6 text-center text-sm text-slate-400">Loading analogs...</p>
      ) : analogs.length === 0 ? (
        <p className="py-6 text-center text-sm text-slate-400">No analogs linked yet.</p>
      ) : (
        <ul className="space-y-2">
          {analogs.map((analog) => (
            <li key={analog.id} className={`rounded-xl ring-1 ring-slate-200 p-3 ${busyId === analog.id ? "opacity-50" : ""}`}>
              <div className="flex items-center gap-3">
                <Thumb url={analog.image} />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-slate-800 truncate">{analog.name}</span>
                    <span className={`badge ${badgeFor(analog.type)}`}>{analog.type}</span>
                    {!analog.is_active && <span className="badge bg-slate-200 text-slate-500">hidden</span>}
                  </div>
                  <div className="text-xs text-slate-400">
                    <span className="font-mono">{analog.sku}</span> · {analog.brand || "No brand"} · stock {analog.stock}
                  </div>
                </div>
                <Btn variant="danger" onClick={() => run(analog.id, () => adminApi.deleteProductAnalog(productId, analog.id))}>
                  ✕
                </Btn>
              </div>

              <div className="flex flex-col sm:flex-row gap-2 mt-3">
                <select
                  className={inputClass + " !py-1.5 sm:w-40"}
                  value={analog.type}
                  onChange={(e) =>
                    run(analog.id, () =>
                      adminApi.updateProductAnalog(productId, analog.id, {
                        type: e.target.value as AnalogType,
                        note: analog.note,
                      }),
                    )
                  }
                >
                  {TYPES.map((t) => <option key={t.value} value={t.value} title={t.hint}>{t.label}</option>)}
                </select>
                <input
                  className={inputClass + " !py-1.5"}
                  placeholder="Note for buyers (optional)"
                  defaultValue={analog.note ?? ""}
                  onBlur={(e) => {
                    const note = e.target.value.trim() || null;
                    if (note === (analog.note ?? null)) return;
                    run(analog.id, () => adminApi.updateProductAnalog(productId, analog.id, { type: analog.type, note }));
                  }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
