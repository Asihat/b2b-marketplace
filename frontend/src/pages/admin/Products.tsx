import { useEffect, useRef, useState } from "react";
import { adminApi, type AdminProduct, type AdminCategory, type Page } from "../../api";
import { PageHeader, Modal, Field, inputClass, Btn } from "../../components/ui";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { ProductPricesModal } from "./ProductPricesModal";

type Draft = Partial<AdminProduct>;
type ProductFilter = "all" | "active" | "hidden" | "b2b";

const EMPTY: Draft = {
  sku: "", name: "", brand: "", description: "", unit: "pcs", base_price: "0", stock: 0, min_order_qty: 1,
  is_b2b_only: false, is_active: true, category_id: null,
};

function pageNumbers(currentPage: number, lastPage: number) {
  const start = Math.max(1, currentPage - 2);
  const end = Math.min(lastPage, currentPage + 2);
  return Array.from({ length: end - start + 1 }, (_, i) => start + i);
}

export function Products() {
  const [products, setProducts] = useState<AdminProduct[]>([]);
  const [meta, setMeta] = useState<Page<AdminProduct> | null>(null);
  const [categories, setCategories] = useState<AdminCategory[]>([]);
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [filter, setFilter] = useState<ProductFilter>("all");
  const [page, setPage] = useState(1);
  const [editing, setEditing] = useState<Draft | null>(null);
  const [imagesText, setImagesText] = useState("");
  const [pricing, setPricing] = useState<AdminProduct | null>(null);
  const [error, setError] = useState("");
  const [listError, setListError] = useState("");
  const [loading, setLoading] = useState(true);
  const requestId = useRef(0);
  const debouncedSearch = useDebouncedValue(search);

  function load() {
    const currentRequest = requestId.current + 1;
    requestId.current = currentRequest;
    setLoading(true);
    setListError("");
    const params = {
      search: debouncedSearch,
      category_id: categoryId || undefined,
      page,
      per_page: 20,
      is_active: filter === "active" ? true : filter === "hidden" ? false : undefined,
      is_b2b_only: filter === "b2b" ? true : undefined,
    };
    adminApi.products(params)
      .then((r) => {
        if (requestId.current !== currentRequest) return;
        setProducts(r.data);
        setMeta(r);
      })
      .catch((e) => {
        if (requestId.current !== currentRequest) return;
        setProducts([]);
        setMeta(null);
        setListError(e instanceof Error ? e.message : "Products could not be loaded.");
      })
      .finally(() => {
        if (requestId.current === currentRequest) setLoading(false);
      });
  }

  function openEditor(draft: Draft) {
    setImagesText((draft.images ?? []).map((i) => i.url).join("\n"));
    setError("");
    setEditing(draft);
  }

  useEffect(() => { adminApi.categories().then(setCategories).catch(() => {}); }, []);
  useEffect(() => { load(); }, [debouncedSearch, categoryId, filter, page]);
  useEffect(() => { setPage(1); }, [debouncedSearch, categoryId, filter]);

  async function save() {
    if (!editing) return;
    setError("");
    const images = imagesText.split("\n").map((s) => s.trim()).filter(Boolean);
    try {
      await adminApi.saveProduct(editing.id ?? null, {
        sku: editing.sku,
        name: editing.name,
        description: editing.description,
        brand: editing.brand,
        unit: editing.unit,
        base_price: Number(editing.base_price),
        stock: Number(editing.stock ?? 0),
        min_order_qty: Number(editing.min_order_qty ?? 1),
        category_id: editing.category_id || null,
        is_b2b_only: !!editing.is_b2b_only,
        is_active: !!editing.is_active,
        images,
      });
      setEditing(null);
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    }
  }

  async function remove(p: AdminProduct) {
    if (!confirm(`Delete ${p.name}?`)) return;
    await adminApi.deleteProduct(p.id);
    load();
  }

  const currentPage = meta?.current_page ?? page;
  const lastPage = meta?.last_page ?? 1;
  const total = meta?.total ?? products.length;
  const activeCount = products.filter((p) => p.is_active).length;
  const hiddenCount = products.filter((p) => !p.is_active).length;
  const b2bCount = products.filter((p) => p.is_b2b_only).length;
  const goToPage = (nextPage: number) => {
    const boundedPage = Math.min(Math.max(nextPage, 1), lastPage);
    if (boundedPage !== currentPage) setPage(boundedPage);
  };

  return (
    <div>
      <PageHeader
        title="Products"
        action={<Btn onClick={() => openEditor({ ...EMPTY })}>+ New product</Btn>}
      />

      <div className="grid gap-3 sm:grid-cols-3 mb-4">
        <div className="card p-4">
          <div className="text-xs font-medium uppercase text-slate-400">Total</div>
          <div className="mt-1 text-2xl font-semibold text-slate-900">{total}</div>
        </div>
        <div className="card p-4">
          <div className="text-xs font-medium uppercase text-slate-400">Visible on this page</div>
          <div className="mt-1 text-2xl font-semibold text-emerald-600">{activeCount}</div>
        </div>
        <div className="card p-4">
          <div className="text-xs font-medium uppercase text-slate-400">B2B / hidden on page</div>
          <div className="mt-1 text-2xl font-semibold text-slate-900">{b2bCount} / {hiddenCount}</div>
        </div>
      </div>

      <div className="flex flex-col xl:flex-row xl:items-center gap-3 mb-4">
        <div className="relative max-w-sm w-full">
          <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
          </svg>
          <input
            placeholder="Search SKU, name, brand..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className={inputClass + " !pl-9"}
          />
        </div>
        <select
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value ? Number(e.target.value) : "")}
          className={inputClass + " max-w-xs"}
        >
          <option value="">All categories</option>
          {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
        <div className="flex flex-wrap gap-2">
          {[
            ["all", "All"],
            ["active", "Active"],
            ["hidden", "Hidden"],
            ["b2b", "B2B only"],
          ].map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setFilter(value as ProductFilter)}
              className={`px-3 py-2 rounded-lg text-sm font-medium transition ${
                filter === value
                  ? "bg-brand-600 text-white shadow-sm shadow-brand-600/20"
                  : "bg-white text-slate-600 ring-1 ring-slate-200 hover:bg-slate-50"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      <div className="card overflow-x-auto">
        <table className="w-full text-sm min-w-[920px]">
          <thead className="bg-slate-50 text-slate-500 text-left">
            <tr>
              <th className="px-4 py-2 w-12"></th>
              <th className="px-4 py-2">SKU</th>
              <th className="px-4 py-2">Product</th>
              <th className="px-4 py-2">Category</th>
              <th className="px-4 py-2 text-right">Price</th>
              <th className="px-4 py-2 text-right">Stock</th>
              <th className="px-4 py-2">Flags</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr><td colSpan={8} className="px-4 py-10 text-center text-slate-400">Loading products...</td></tr>
            )}
            {!loading && listError && (
              <tr><td colSpan={8} className="px-4 py-10 text-center text-rose-600">{listError}</td></tr>
            )}
            {!loading && !listError && products.map((p) => (
              <tr key={p.id} className="border-t border-slate-100">
                <td className="px-2 py-2">
                  <div className="w-11 h-11 rounded-lg bg-slate-100 overflow-hidden">
                    {p.images?.[0] ? (
                      <img src={p.images[0].url} alt="" className="w-full h-full object-cover" />
                    ) : (
                      <div className="w-full h-full grid place-items-center text-slate-300 text-lg">◆</div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2 font-mono text-xs text-slate-500">{p.sku}</td>
                <td className="px-4 py-2">
                  <div className="font-medium text-slate-800">{p.name}</div>
                  <div className="text-xs text-slate-400">{p.brand || "No brand"} · MOQ {p.min_order_qty}</div>
                </td>
                <td className="px-4 py-2 text-slate-500">{p.category?.name ?? "—"}</td>
                <td className="px-4 py-2 text-right font-medium">{Number(p.base_price).toFixed(2)}</td>
                <td className="px-4 py-2 text-right">
                  <span className={p.stock <= 0 ? "text-rose-600 font-medium" : p.stock <= p.min_order_qty ? "text-amber-600 font-medium" : ""}>
                    {p.stock}
                  </span>
                </td>
                <td className="px-4 py-2 text-xs space-x-1">
                  {p.is_active ? <span className="badge bg-emerald-50 text-emerald-700">active</span> : <span className="badge bg-slate-200 text-slate-500">hidden</span>}
                  {p.is_b2b_only && <span className="badge bg-amber-100 text-amber-700">B2B</span>}
                </td>
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap">
                  <Btn variant="ghost" onClick={() => setPricing(p)}>Prices</Btn>
                  <Btn variant="ghost" onClick={() => openEditor({ ...p })}>Edit</Btn>
                  <Btn variant="danger" onClick={() => remove(p)}>Delete</Btn>
                </td>
              </tr>
            ))}
            {!loading && !listError && products.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-10 text-center text-slate-400">No products match these filters.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {lastPage > 1 && (
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mt-4">
          <p className="text-sm text-slate-500">Page {currentPage} of {lastPage} · {total} products</p>
          <div className="flex items-center gap-1.5">
            <button type="button" onClick={() => goToPage(currentPage - 1)} disabled={currentPage <= 1} className="btn-soft !px-3 !py-2 disabled:opacity-40">
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m15 18-6-6 6-6" /></svg>
            </button>
            {pageNumbers(currentPage, lastPage).map((pageNumber) => (
              <button
                key={pageNumber}
                type="button"
                onClick={() => goToPage(pageNumber)}
                className={`min-w-10 h-10 rounded-xl text-sm font-medium transition ${
                  currentPage === pageNumber
                    ? "bg-brand-600 text-white shadow-sm shadow-brand-600/20"
                    : "bg-white text-slate-600 ring-1 ring-slate-200 hover:bg-slate-50 hover:ring-slate-300"
                }`}
              >
                {pageNumber}
              </button>
            ))}
            <button type="button" onClick={() => goToPage(currentPage + 1)} disabled={currentPage >= lastPage} className="btn-soft !px-3 !py-2 disabled:opacity-40">
              <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m9 18 6-6-6-6" /></svg>
            </button>
          </div>
        </div>
      )}

      {editing && (
        <Modal title={editing.id ? "Edit product" : "New product"} onClose={() => setEditing(null)}>
          <div className="grid grid-cols-2 gap-3">
            <Field label="SKU">
              <input className={inputClass} value={editing.sku ?? ""} onChange={(e) => setEditing({ ...editing, sku: e.target.value })} />
            </Field>
            <Field label="Brand">
              <input className={inputClass} value={editing.brand ?? ""} onChange={(e) => setEditing({ ...editing, brand: e.target.value })} />
            </Field>
          </div>
          <Field label="Name">
            <input className={inputClass} value={editing.name ?? ""} onChange={(e) => setEditing({ ...editing, name: e.target.value })} />
          </Field>
          <Field label="Description">
            <textarea rows={3} className={inputClass} value={editing.description ?? ""} onChange={(e) => setEditing({ ...editing, description: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <Field label="Base price">
              <input type="number" step="0.01" className={inputClass} value={editing.base_price ?? ""} onChange={(e) => setEditing({ ...editing, base_price: e.target.value })} />
            </Field>
            <Field label="Stock">
              <input type="number" className={inputClass} value={editing.stock ?? 0} onChange={(e) => setEditing({ ...editing, stock: Number(e.target.value) })} />
            </Field>
            <Field label="Min order qty">
              <input type="number" className={inputClass} value={editing.min_order_qty ?? 1} onChange={(e) => setEditing({ ...editing, min_order_qty: Number(e.target.value) })} />
            </Field>
            <Field label="Unit">
              <input className={inputClass} value={editing.unit ?? ""} onChange={(e) => setEditing({ ...editing, unit: e.target.value })} />
            </Field>
          </div>
          <Field label="Category">
            <select className={inputClass} value={editing.category_id ?? ""} onChange={(e) => setEditing({ ...editing, category_id: e.target.value ? Number(e.target.value) : null })}>
              <option value="">— none —</option>
              {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </Field>
          <Field label="Image URLs (one per line — first is primary)">
            <textarea
              rows={3}
              className={inputClass + " font-mono text-xs"}
              placeholder="https://…/image1.jpg&#10;https://…/image2.jpg"
              value={imagesText}
              onChange={(e) => setImagesText(e.target.value)}
            />
          </Field>
          {imagesText.trim() && (
            <div className="flex gap-2 mb-3">
              {imagesText.split("\n").map((s) => s.trim()).filter(Boolean).slice(0, 5).map((url, i) => (
                <div key={i} className="w-12 h-12 rounded bg-slate-100 overflow-hidden">
                  <img src={url} alt="" className="w-full h-full object-cover" />
                </div>
              ))}
            </div>
          )}
          <div className="flex gap-4 mb-4 text-sm">
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={!!editing.is_b2b_only} onChange={(e) => setEditing({ ...editing, is_b2b_only: e.target.checked })} />
              B2B only
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={!!editing.is_active} onChange={(e) => setEditing({ ...editing, is_active: e.target.checked })} />
              Active
            </label>
          </div>
          {error && <p className="text-sm text-red-600 mb-3">{error}</p>}
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={() => setEditing(null)}>Cancel</Btn>
            <Btn onClick={save}>Save</Btn>
          </div>
        </Modal>
      )}

      {pricing && <ProductPricesModal product={pricing} onClose={() => setPricing(null)} />}
    </div>
  );
}
