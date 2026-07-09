import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { adminApi, type AdminProduct, type AdminCategory, type Page } from "../../api";
import { PageHeader, inputClass, Btn } from "../../components/ui";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { useStore } from "../../store";

type ProductFilter = "all" | "active" | "hidden" | "b2b";

function pageNumbers(currentPage: number, lastPage: number) {
  const start = Math.max(1, currentPage - 2);
  const end = Math.min(lastPage, currentPage + 2);
  return Array.from({ length: end - start + 1 }, (_, i) => start + i);
}

export function Products() {
  const { settings } = useStore();
  const navigate = useNavigate();
  const [products, setProducts] = useState<AdminProduct[]>([]);
  const [meta, setMeta] = useState<Page<AdminProduct> | null>(null);
  const [categories, setCategories] = useState<AdminCategory[]>([]);
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [filter, setFilter] = useState<ProductFilter>("all");
  const [page, setPage] = useState(1);
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

  useEffect(() => { adminApi.categories().then(setCategories).catch(() => {}); }, []);
  useEffect(() => { load(); }, [debouncedSearch, categoryId, filter, page]);
  useEffect(() => { setPage(1); }, [debouncedSearch, categoryId, filter]);

  async function remove(p: AdminProduct) {
    if (!confirm(`Delete ${p.name}?`)) return;
    await adminApi.deleteProduct(p.id);
    load();
  }

  const currentPage = meta?.current_page ?? page;
  const lastPage = meta?.last_page ?? 1;
  const total = meta?.total ?? products.length;
  const showSeller = settings.show_company_names;
  const tableColSpan = showSeller ? 9 : 8;
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
        action={<Link to="/admin/products/new" className="btn-primary">+ New product</Link>}
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
              {showSeller && <th className="px-4 py-2">Seller</th>}
              <th className="px-4 py-2 text-right">Price</th>
              <th className="px-4 py-2 text-right">Stock</th>
              <th className="px-4 py-2">Flags</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {loading && (
              <tr><td colSpan={tableColSpan} className="px-4 py-10 text-center text-slate-400">Loading products...</td></tr>
            )}
            {!loading && listError && (
              <tr><td colSpan={tableColSpan} className="px-4 py-10 text-center text-rose-600">{listError}</td></tr>
            )}
            {!loading && !listError && products.map((p) => (
              <tr
                key={p.id}
                onClick={() => navigate(`/admin/products/${p.id}`)}
                className="border-t border-slate-100 hover:bg-slate-50/70 cursor-pointer transition"
              >
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
                  <div className="text-xs text-slate-400">
                    {p.brand || "No brand"} · MOQ {p.min_order_qty}
                    {!!p.analogs_count && ` · ${p.analogs_count} analog${p.analogs_count === 1 ? "" : "s"}`}
                  </div>
                </td>
                <td className="px-4 py-2 text-slate-500">{p.category?.name ?? "—"}</td>
                {showSeller && <td className="px-4 py-2 text-slate-500">{p.company?.name ?? "—"}</td>}
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
                <td className="px-4 py-2 text-right space-x-1 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                  <Link to={`/admin/products/${p.id}`} className="btn-ghost !px-3 !py-1.5">Edit</Link>
                  <Btn variant="danger" onClick={() => remove(p)}>Delete</Btn>
                </td>
              </tr>
            ))}
            {!loading && !listError && products.length === 0 && (
              <tr><td colSpan={tableColSpan} className="px-4 py-10 text-center text-slate-400">No products match these filters.</td></tr>
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
    </div>
  );
}
