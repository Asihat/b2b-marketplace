import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { api, type Analog, type Product } from "../api";
import { useStore } from "../store";

export function ProductDetail() {
  const { slug } = useParams();
  const { currency, locale, addToCart } = useStore();
  const [product, setProduct] = useState<Product | null>(null);
  const [analogs, setAnalogs] = useState<Analog[]>([]);
  const [qty, setQty] = useState(1);
  const [activeImg, setActiveImg] = useState(0);
  const [added, setAdded] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!slug) return;
    setError("");
    setActiveImg(0);
    api
      .product(slug, { currency, lang: locale, qty })
      .then((p) => {
        setProduct(p);
        setQty(p.min_order_qty);
        return api.analogs(p.id, { currency, lang: locale });
      })
      .then(setAnalogs)
      .catch((e) => setError(e.message));
  }, [slug, currency, locale]);

  useEffect(() => {
    if (product) {
      api.product(product.slug, { currency, lang: locale, qty }).then(setProduct).catch(() => {});
    }
  }, [qty]);

  function handleAdd() {
    if (!product) return;
    addToCart(product, qty);
    setAdded(true);
    setTimeout(() => setAdded(false), 1600);
  }

  if (error) return <div className="card p-8 text-rose-600">{error}</div>;
  if (!product) {
    return (
      <div className="grid md:grid-cols-2 gap-8">
        <div className="aspect-square skeleton" />
        <div className="space-y-4">
          <div className="h-7 w-2/3 skeleton" />
          <div className="h-4 w-1/3 skeleton" />
          <div className="h-24 w-full skeleton" />
        </div>
      </div>
    );
  }

  const images = product.images ?? [];

  return (
    <div>
      <Link to="/" className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-brand-600 transition mb-5">
        <span aria-hidden>←</span> Back to catalog
      </Link>

      <div className="grid md:grid-cols-2 gap-8">
        {/* Gallery */}
        <div>
          <div className="card aspect-square overflow-hidden">
            {images.length > 0 ? (
              <img src={images[activeImg]?.url} alt={product.name} className="w-full h-full object-cover" />
            ) : product.image ? (
              <img src={product.image} alt={product.name} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full grid place-items-center text-slate-300 text-6xl">◆</div>
            )}
          </div>
          {images.length > 1 && (
            <div className="flex gap-2.5 mt-3">
              {images.map((img, i) => (
                <button
                  key={i}
                  onClick={() => setActiveImg(i)}
                  className={`w-16 h-16 rounded-xl overflow-hidden ring-2 transition ${
                    i === activeImg ? "ring-brand-500" : "ring-transparent hover:ring-slate-200"
                  }`}
                >
                  <img src={img.url} alt={img.alt ?? ""} className="w-full h-full object-cover" />
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Details */}
        <div className="space-y-6">
          <div>
            <div className="flex items-center gap-2 text-xs">
              <span className="font-medium uppercase tracking-wide text-slate-400">{product.brand}</span>
              {product.is_b2b_only && <span className="badge bg-amber-100 text-amber-700">Business only</span>}
            </div>
            <h1 className="mt-2 text-2xl sm:text-3xl font-bold text-slate-900">{product.name}</h1>
            <div className="mt-1 text-sm text-slate-400 font-mono">{product.sku}</div>
            <p className="mt-4 text-slate-600 leading-relaxed">{product.description}</p>
          </div>

          <div className="card p-5">
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-bold text-slate-900">{product.price.formatted}</span>
              <span className="text-xs text-slate-400">/ {product.unit} · {currency}</span>
            </div>
            <div className="mt-2 flex items-center gap-2 text-sm">
              <span className={`inline-flex items-center gap-1.5 ${product.stock > 0 ? "text-emerald-600" : "text-rose-500"}`}>
                <span className={`w-1.5 h-1.5 rounded-full ${product.stock > 0 ? "bg-emerald-500" : "bg-rose-500"}`} />
                {product.stock > 0 ? `${product.stock} in stock` : "Out of stock"}
              </span>
              {product.min_order_qty > 1 && <span className="text-slate-400">· min order {product.min_order_qty}</span>}
            </div>

            <div className="flex items-center gap-3 mt-5">
              <div className="flex items-center rounded-xl ring-1 ring-slate-200 bg-white overflow-hidden">
                <button
                  onClick={() => setQty((q) => Math.max(product.min_order_qty, q - 1))}
                  className="px-3 py-2.5 text-slate-500 hover:bg-slate-50"
                >−</button>
                <input
                  type="number"
                  min={product.min_order_qty}
                  value={qty}
                  onChange={(e) => setQty(Math.max(product.min_order_qty, Number(e.target.value)))}
                  className="w-14 text-center py-2.5 focus:outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                />
                <button onClick={() => setQty((q) => q + 1)} className="px-3 py-2.5 text-slate-500 hover:bg-slate-50">+</button>
              </div>
              <button
                disabled={product.stock <= 0}
                onClick={handleAdd}
                className={`btn flex-1 text-white transition ${added ? "bg-emerald-600" : "btn-primary"}`}
              >
                {added ? "✓ Added to cart" : "Add to cart"}
              </button>
            </div>
            <p className="text-xs text-slate-400 mt-3">
              Price updates automatically for B2B volume tiers as quantity changes.
            </p>
          </div>
        </div>
      </div>

      {/* Analogs */}
      <section className="mt-12">
        <h2 className="text-lg font-semibold text-slate-900 mb-4">Analogs & cross-references</h2>
        {analogs.length === 0 ? (
          <p className="text-slate-400 text-sm">No analogs listed for this product.</p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {analogs.map((a) => (
              <Link key={a.id} to={`/products/${a.slug}`} className="card card-hover p-3 flex gap-3">
                <div className="w-16 h-16 shrink-0 rounded-lg overflow-hidden bg-slate-100">
                  {a.image ? (
                    <img src={a.image} alt={a.name} loading="lazy" className="w-full h-full object-cover" />
                  ) : (
                    <div className="w-full h-full grid place-items-center text-slate-300">◆</div>
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium text-slate-800 truncate">{a.name}</span>
                    {a.relation && <span className="badge bg-brand-50 text-brand-600 shrink-0">{a.relation.type}</span>}
                  </div>
                  <div className="text-xs text-slate-400 mt-0.5">{a.brand} · {a.sku}</div>
                  <div className="text-sm font-semibold text-slate-900 mt-1">{a.price.formatted}</div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
