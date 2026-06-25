import { useEffect, useRef, useState } from "react";
import { api, type Category, type Paginated, type Product } from "../api";
import { useStore } from "../store";
import { useT } from "../i18n";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { ProductCard } from "../components/ProductCard";

type SortOption = "name" | "price_asc" | "price_desc" | "newest" | "popular";

function CardSkeleton() {
  return (
    <div className="card overflow-hidden">
      <div className="aspect-square skeleton rounded-none" />
      <div className="p-4 space-y-2">
        <div className="h-2.5 w-1/3 skeleton" />
        <div className="h-4 w-3/4 skeleton" />
        <div className="h-5 w-1/2 skeleton mt-3" />
      </div>
    </div>
  );
}

function pageNumbers(currentPage: number, lastPage: number) {
  const start = Math.max(1, currentPage - 2);
  const end = Math.min(lastPage, currentPage + 2);
  return Array.from({ length: end - start + 1 }, (_, i) => start + i);
}

export function Catalog() {
  const { currency, locale } = useStore();
  const t = useT();
  const [products, setProducts] = useState<Product[]>([]);
  const [meta, setMeta] = useState<Paginated<Product>["meta"] | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [minPrice, setMinPrice] = useState("");
  const [maxPrice, setMaxPrice] = useState("");
  const [sort, setSort] = useState<SortOption>("name");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const requestId = useRef(0);
  const debouncedSearch = useDebouncedValue(search);
  const debouncedMinPrice = useDebouncedValue(minPrice);
  const debouncedMaxPrice = useDebouncedValue(maxPrice);

  useEffect(() => {
    api.categories().then(setCategories).catch(() => {});
  }, []);

  useEffect(() => {
    const currentRequest = requestId.current + 1;
    requestId.current = currentRequest;
    setLoading(true);
    const sortParams = {
      name: { sort: "name", direction: "asc" },
      price_asc: { sort: "base_price", direction: "asc" },
      price_desc: { sort: "base_price", direction: "desc" },
      newest: { sort: "created_at", direction: "desc" },
      popular: { sort: "popular", direction: "desc" },
    }[sort];
    api
      .products({
        currency,
        lang: locale,
        search: debouncedSearch,
        category_id: categoryId || undefined,
        min_price: debouncedMinPrice || undefined,
        max_price: debouncedMaxPrice || undefined,
        page,
        per_page: 24,
        ...sortParams,
      })
      .then((res) => {
        if (requestId.current !== currentRequest) return;
        setProducts(res.data);
        setMeta(res.meta);
      })
      .catch(() => {
        if (requestId.current !== currentRequest) return;
        setProducts([]);
        setMeta(null);
      })
      .finally(() => {
        if (requestId.current === currentRequest) setLoading(false);
      });
  }, [currency, locale, debouncedSearch, categoryId, debouncedMinPrice, debouncedMaxPrice, sort, page]);

  useEffect(() => {
    setPage(1);
  }, [currency, locale, debouncedSearch, categoryId, debouncedMinPrice, debouncedMaxPrice, sort]);

  const totalProducts = meta?.total ?? products.length;
  const currentPage = meta?.current_page ?? page;
  const lastPage = meta?.last_page ?? 1;
  const hasPagination = lastPage > 1;
  const hasAdvancedFilters = minPrice !== "" || maxPrice !== "" || sort !== "name";
  const clearAdvancedFilters = () => {
    setMinPrice("");
    setMaxPrice("");
    setSort("name");
  };
  const goToPage = (nextPage: number) => {
    const boundedPage = Math.min(Math.max(nextPage, 1), lastPage);
    if (boundedPage !== currentPage) {
      setPage(boundedPage);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
  };

  return (
    <div>
      {/* Hero */}
      <section className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-brand-600 via-brand-600 to-violet-700 text-white px-6 sm:px-10 py-10 sm:py-14 mb-8">
        <div className="absolute -right-16 -top-16 w-64 h-64 rounded-full bg-white/10 blur-2xl" />
        <div className="absolute right-24 bottom-0 w-40 h-40 rounded-full bg-violet-400/20 blur-2xl" />
        <div className="relative max-w-xl">
          <span className="badge bg-white/15 text-white/90 ring-1 ring-white/20">{t("catalog.heroBadge")}</span>
          <h1 className="mt-3 text-3xl sm:text-4xl font-bold tracking-tight text-white">
            {t("catalog.heroTitle")}
          </h1>
          <p className="mt-3 text-white/80 text-sm sm:text-base">
            {t("catalog.heroSubtitle", { count: totalProducts > 0 ? totalProducts : "100+" })}
          </p>
        </div>
      </section>

      {/* Search + filters */}
      <div className="flex flex-col gap-4 mb-6">
        <div className="grid gap-3 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(180px,220px)]">
          <div className="relative">
            <svg className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" />
            </svg>
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t("catalog.searchPlaceholder")}
              className="input !pl-10 !py-3 shadow-sm"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <input
              type="number"
              min="0"
              inputMode="decimal"
              value={minPrice}
              onChange={(e) => setMinPrice(e.target.value)}
              placeholder={t("catalog.minPrice")}
              className="input !py-3 shadow-sm"
            />
            <input
              type="number"
              min="0"
              inputMode="decimal"
              value={maxPrice}
              onChange={(e) => setMaxPrice(e.target.value)}
              placeholder={t("catalog.maxPrice")}
              className="input !py-3 shadow-sm"
            />
          </div>

          <select
            value={sort}
            onChange={(e) => setSort(e.target.value as SortOption)}
            className="input !py-3 shadow-sm"
            aria-label={t("catalog.sortLabel")}
          >
            <option value="name">{t("catalog.sortName")}</option>
            <option value="price_asc">{t("catalog.sortPriceAsc")}</option>
            <option value="price_desc">{t("catalog.sortPriceDesc")}</option>
            <option value="newest">{t("catalog.sortNewest")}</option>
            <option value="popular">{t("catalog.sortPopular")}</option>
          </select>
        </div>

        <div className="flex gap-2 overflow-x-auto pb-1 -mx-1 px-1">
          <button
            onClick={() => setCategoryId("")}
            className={`badge !text-xs !px-3 !py-1.5 whitespace-nowrap transition ${
              categoryId === "" ? "bg-brand-600 text-white" : "bg-white ring-1 ring-slate-200 text-slate-600 hover:ring-slate-300"
            }`}
          >
            {t("common.all")}
          </button>
          {categories.map((c) => (
            <button
              key={c.id}
              onClick={() => setCategoryId(c.id)}
              className={`badge !text-xs !px-3 !py-1.5 whitespace-nowrap transition ${
                categoryId === c.id ? "bg-brand-600 text-white" : "bg-white ring-1 ring-slate-200 text-slate-600 hover:ring-slate-300"
              }`}
            >
              {c.name}
            </button>
          ))}
          {hasAdvancedFilters && (
            <button
              onClick={clearAdvancedFilters}
              className="badge !text-xs !px-3 !py-1.5 whitespace-nowrap bg-slate-100 text-slate-500 hover:bg-slate-200 transition"
            >
              {t("catalog.clearFilters")}
            </button>
          )}
        </div>
      </div>

      {/* Grid */}
      {loading ? (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-5">
          {Array.from({ length: 8 }).map((_, i) => <CardSkeleton key={i} />)}
        </div>
      ) : products.length === 0 ? (
        <div className="card py-20 text-center">
          <div className="text-5xl mb-3">🔍</div>
          <p className="text-slate-500">{t("catalog.empty")}</p>
        </div>
      ) : (
        <div className="space-y-6">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-5">
            {products.map((p) => <ProductCard key={p.id} product={p} />)}
          </div>

          {hasPagination && (
            <nav className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3" aria-label={t("catalog.paginationLabel")}>
              <p className="text-sm text-slate-500">
                {t("catalog.pageSummary", { page: currentPage, pages: lastPage, total: totalProducts })}
              </p>
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => goToPage(currentPage - 1)}
                  disabled={currentPage <= 1}
                  className="btn-soft !px-3 !py-2 disabled:opacity-40"
                  aria-label={t("catalog.previousPage")}
                >
                  <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="m15 18-6-6 6-6" />
                  </svg>
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
                    aria-current={currentPage === pageNumber ? "page" : undefined}
                  >
                    {pageNumber}
                  </button>
                ))}
                <button
                  type="button"
                  onClick={() => goToPage(currentPage + 1)}
                  disabled={currentPage >= lastPage}
                  className="btn-soft !px-3 !py-2 disabled:opacity-40"
                  aria-label={t("catalog.nextPage")}
                >
                  <svg className="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="m9 18 6-6-6-6" />
                  </svg>
                </button>
              </div>
            </nav>
          )}
        </div>
      )}
    </div>
  );
}
