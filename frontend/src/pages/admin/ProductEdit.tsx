import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { adminApi, type AdminCategory, type AdminProduct, type Currency } from "../../api";
import { Btn, Field, inputClass } from "../../components/ui";
import { PriceTiers } from "../../components/admin/PriceTiers";
import { AnalogLinks } from "../../components/admin/AnalogLinks";

type Draft = {
  sku: string;
  name: string;
  brand: string;
  description: string;
  unit: string;
  base_price: string;
  stock: number;
  min_order_qty: number;
  category_id: number | null;
  is_b2b_only: boolean;
  is_active: boolean;
};

const EMPTY: Draft = {
  sku: "", name: "", brand: "", description: "", unit: "pcs", base_price: "0",
  stock: 0, min_order_qty: 1, category_id: null, is_b2b_only: false, is_active: true,
};

const toDraft = (p: AdminProduct): Draft => ({
  sku: p.sku,
  name: p.name,
  brand: p.brand ?? "",
  description: p.description ?? "",
  unit: p.unit ?? "pcs",
  base_price: Number(p.base_price).toString(),
  stock: p.stock,
  min_order_qty: p.min_order_qty,
  category_id: p.category_id,
  is_b2b_only: p.is_b2b_only,
  is_active: p.is_active,
});

export function ProductEdit() {
  const { id } = useParams();
  const navigate = useNavigate();
  const isNew = id === "new";
  const productId = isNew ? null : Number(id);

  const [product, setProduct] = useState<AdminProduct | null>(null);
  const [draft, setDraft] = useState<Draft>(EMPTY);
  const [images, setImages] = useState<string[]>([]);
  const [categories, setCategories] = useState<AdminCategory[]>([]);
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [analogCount, setAnalogCount] = useState(0);
  const [priceCount, setPriceCount] = useState(0);

  const [loading, setLoading] = useState(!isNew);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    adminApi.categories().then(setCategories).catch(() => {});
    adminApi.currencies().then(setCurrencies).catch(() => {});
  }, []);

  useEffect(() => {
    if (productId === null) return;
    setLoading(true);
    adminApi.product(productId)
      .then((p) => {
        setProduct(p);
        setDraft(toDraft(p));
        setImages((p.images ?? []).map((i) => i.url));
        setAnalogCount(p.analogs_count ?? 0);
        setPriceCount(p.prices?.length ?? 0);
      })
      .catch((e) => setError(e instanceof Error ? e.message : "Product could not be loaded."))
      .finally(() => setLoading(false));
  }, [productId]);

  const set = <K extends keyof Draft>(key: K, value: Draft[K]) => {
    setDraft((d) => ({ ...d, [key]: value }));
    setSaved(false);
  };

  const moveImage = (from: number, to: number) => {
    if (to < 0 || to >= images.length) return;
    const next = [...images];
    [next[from], next[to]] = [next[to], next[from]];
    setImages(next);
    setSaved(false);
  };

  async function save() {
    setSaving(true);
    setError("");
    try {
      const result = await adminApi.saveProduct(productId, {
        ...draft,
        brand: draft.brand.trim() || null,
        description: draft.description.trim() || null,
        base_price: Number(draft.base_price),
        stock: Number(draft.stock),
        min_order_qty: Number(draft.min_order_qty),
        images: images.map((u) => u.trim()).filter(Boolean),
      });

      if (isNew) {
        navigate(`/admin/products/${result.id}`, { replace: true });
        return;
      }
      setProduct(result);
      setDraft(toDraft(result));
      setImages((result.images ?? []).map((i) => i.url));
      setSaved(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  async function remove() {
    if (!product || !confirm(`Delete ${product.name}? This also removes its prices and analog links.`)) return;
    await adminApi.deleteProduct(product.id);
    navigate("/admin/products");
  }

  if (loading) return <p className="py-20 text-center text-slate-400">Loading product...</p>;
  if (!isNew && !product) return <p className="py-20 text-center text-rose-600">{error || "Product not found."}</p>;

  const basePrice = Number(draft.base_price) || 0;

  return (
    <div className="max-w-6xl">
      <Link to="/admin/products" className="text-sm text-slate-500 hover:text-slate-800 transition">← Products</Link>

      <div className="flex flex-wrap items-start justify-between gap-4 mt-2 mb-6">
        <div className="min-w-0">
          <h1 className="text-xl font-bold tracking-tight text-slate-900 truncate">
            {isNew ? "New product" : product!.name}
          </h1>
          {!isNew && (
            <div className="flex items-center gap-2 mt-1 text-sm text-slate-400">
              <span className="font-mono">{product!.sku}</span>
              {product!.is_active
                ? <span className="badge bg-emerald-50 text-emerald-700">active</span>
                : <span className="badge bg-slate-200 text-slate-500">hidden</span>}
              {product!.is_b2b_only && <span className="badge bg-amber-100 text-amber-700">B2B only</span>}
              <Link to={`/products/${product!.slug}`} className="text-brand-600 hover:text-brand-700">View in store ↗</Link>
            </div>
          )}
        </div>
        <div className="flex items-center gap-2">
          {saved && <span className="text-sm text-emerald-600">Saved</span>}
          {!isNew && <Btn variant="danger" onClick={remove}>Delete</Btn>}
          <Btn onClick={save} disabled={saving || !draft.sku.trim() || !draft.name.trim()}>
            {saving ? "Saving..." : isNew ? "Create product" : "Save changes"}
          </Btn>
        </div>
      </div>

      {error && <p className="text-sm text-rose-600 bg-rose-50 rounded-lg px-3 py-2 mb-4">{error}</p>}

      <div className="grid lg:grid-cols-3 gap-6 items-start">
        <div className="lg:col-span-2 space-y-6">
          <div className="card p-5">
            <h2 className="text-base font-semibold text-slate-900 mb-4">Details</h2>
            <div className="grid sm:grid-cols-2 gap-x-3">
              <Field label="SKU">
                <input className={inputClass} value={draft.sku} onChange={(e) => set("sku", e.target.value)} />
              </Field>
              <Field label="Brand">
                <input className={inputClass} value={draft.brand} onChange={(e) => set("brand", e.target.value)} />
              </Field>
            </div>
            <Field label="Name">
              <input className={inputClass} value={draft.name} onChange={(e) => set("name", e.target.value)} />
            </Field>
            <Field label="Description">
              <textarea rows={4} className={inputClass} value={draft.description} onChange={(e) => set("description", e.target.value)} />
            </Field>
            <Field label="Category">
              <select
                className={inputClass}
                value={draft.category_id ?? ""}
                onChange={(e) => set("category_id", e.target.value ? Number(e.target.value) : null)}
              >
                <option value="">— none —</option>
                {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </Field>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-x-3">
              <Field label="Base price">
                <input type="number" step="0.01" min={0} className={inputClass} value={draft.base_price} onChange={(e) => set("base_price", e.target.value)} />
              </Field>
              <Field label="Stock">
                <input type="number" min={0} className={inputClass} value={draft.stock} onChange={(e) => set("stock", Number(e.target.value))} />
              </Field>
              <Field label="Min order qty">
                <input type="number" min={1} className={inputClass} value={draft.min_order_qty} onChange={(e) => set("min_order_qty", Number(e.target.value))} />
              </Field>
              <Field label="Unit">
                <input className={inputClass} value={draft.unit} onChange={(e) => set("unit", e.target.value)} />
              </Field>
            </div>
            <div className="flex gap-5 text-sm">
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={draft.is_active} onChange={(e) => set("is_active", e.target.checked)} />
                Active
              </label>
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={draft.is_b2b_only} onChange={(e) => set("is_b2b_only", e.target.checked)} />
                B2B only
              </label>
            </div>
          </div>

          <div className="card p-5">
            <h2 className="text-base font-semibold text-slate-900">Images</h2>
            <p className="text-sm text-slate-500 mt-0.5 mb-4">The first image is the one buyers see in the catalog.</p>

            <div className="space-y-2">
              {images.map((url, i) => (
                <div key={i} className="flex items-center gap-2">
                  <div className="w-12 h-12 rounded-lg bg-slate-100 overflow-hidden shrink-0">
                    {url.trim() && <img src={url} alt="" className="w-full h-full object-cover" />}
                  </div>
                  <input
                    className={inputClass + " font-mono text-xs"}
                    placeholder="https://…/image.jpg"
                    value={url}
                    onChange={(e) => { setImages(images.map((u, j) => (j === i ? e.target.value : u))); setSaved(false); }}
                  />
                  {i === 0 && <span className="badge bg-brand-50 text-brand-700 shrink-0">primary</span>}
                  <button type="button" onClick={() => moveImage(i, i - 1)} disabled={i === 0} className="btn-soft !px-2 !py-1.5 disabled:opacity-30">↑</button>
                  <button type="button" onClick={() => moveImage(i, i + 1)} disabled={i === images.length - 1} className="btn-soft !px-2 !py-1.5 disabled:opacity-30">↓</button>
                  <Btn variant="danger" onClick={() => { setImages(images.filter((_, j) => j !== i)); setSaved(false); }}>✕</Btn>
                </div>
              ))}
              {images.length === 0 && <p className="py-4 text-center text-sm text-slate-400">No images yet.</p>}
            </div>

            <div className="mt-3">
              <Btn variant="ghost" onClick={() => setImages([...images, ""])}>+ Add image URL</Btn>
            </div>
          </div>

          {productId !== null && currencies.length > 0 && (
            <PriceTiers
              productId={productId}
              basePrice={basePrice}
              currencies={currencies}
              initial={product!.prices ?? []}
              onSaved={(rows) => setPriceCount(rows.length)}
            />
          )}
        </div>

        <div className="space-y-6">
          {productId === null ? (
            <div className="card p-5 text-sm text-slate-500">
              Create the product first — prices and analogs can be set once it exists.
            </div>
          ) : (
            <>
              <div className="card p-5">
                <h2 className="text-base font-semibold text-slate-900 mb-3">At a glance</h2>
                <dl className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Price overrides</dt>
                    <dd className="font-medium text-slate-800">{priceCount}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Analogs</dt>
                    <dd className="font-medium text-slate-800">{analogCount}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Seller</dt>
                    <dd className="font-medium text-slate-800">{product!.company?.name ?? "—"}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-500">Stock</dt>
                    <dd className={`font-medium ${product!.stock <= 0 ? "text-rose-600" : "text-slate-800"}`}>
                      {product!.stock} {product!.unit}
                    </dd>
                  </div>
                </dl>
              </div>

              <AnalogLinks productId={productId} onCountChange={setAnalogCount} />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
