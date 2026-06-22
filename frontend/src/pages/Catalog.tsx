import { useEffect, useState } from "react";
import { api, type Category, type Product } from "../api";
import { useStore } from "../store";
import { useT } from "../i18n";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { ProductCard } from "../components/ProductCard";

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

export function Catalog() {
  const { currency, locale } = useStore();
  const t = useT();
  const [products, setProducts] = useState<Product[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [search, setSearch] = useState("");
  const [categoryId, setCategoryId] = useState<number | "">("");
  const [loading, setLoading] = useState(true);
  const debouncedSearch = useDebouncedValue(search);

  useEffect(() => {
    api.categories().then(setCategories).catch(() => {});
  }, []);

  useEffect(() => {
    setLoading(true);
    api
      .products({ currency, lang: locale, search: debouncedSearch, category_id: categoryId || undefined, per_page: 24 })
      .then((res) => setProducts(res.data))
      .catch(() => setProducts([]))
      .finally(() => setLoading(false));
  }, [currency, locale, debouncedSearch, categoryId]);

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
            {t("catalog.heroSubtitle", { count: products.length > 0 ? `${products.length}+` : "100+" })}
          </p>
        </div>
      </section>

      {/* Search + filters */}
      <div className="flex flex-col gap-4 mb-6">
        <div className="relative max-w-xl">
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
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-5">
          {products.map((p) => <ProductCard key={p.id} product={p} />)}
        </div>
      )}
    </div>
  );
}
