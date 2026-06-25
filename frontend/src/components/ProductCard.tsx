import { Link } from "react-router-dom";
import type { Product } from "../api";
import { useStore } from "../store";
import { useT } from "../i18n";

export function ProductCard({ product }: { product: Product }) {
  const { addToCart, settings } = useStore();
  const t = useT();
  const showSeller = settings.show_company_names && product.company;

  return (
    <div className="card card-hover group flex flex-col overflow-hidden">
      <Link to={`/products/${product.slug}`} className="relative block aspect-square overflow-hidden bg-slate-100">
        {product.image ? (
          <img
            src={product.image}
            alt={product.name}
            loading="lazy"
            className="w-full h-full object-cover transition duration-500 ease-out group-hover:scale-[1.06]"
          />
        ) : (
          <div className="w-full h-full grid place-items-center text-slate-300 text-4xl">◆</div>
        )}
        {product.is_b2b_only && (
          <span className="badge absolute top-2.5 left-2.5 bg-amber-100/90 text-amber-700 backdrop-blur-sm">
            {t("product.b2bOnly")}
          </span>
        )}
        {product.stock <= 0 && (
          <span className="badge absolute top-2.5 right-2.5 bg-slate-900/80 text-white backdrop-blur-sm">
            {t("product.soldOut")}
          </span>
        )}
      </Link>

      <div className="flex flex-col flex-1 p-4">
        <div className="text-[11px] font-medium uppercase tracking-wide text-slate-400">
          {product.brand ?? "—"}
        </div>
        {showSeller && (
          <div className="mt-1 flex items-center gap-1.5 text-[11px] text-slate-500">
            <span className="text-slate-400">{t("product.seller")}</span>
            <span className="font-medium text-slate-700 truncate">{product.company?.name}</span>
            {product.company?.is_verified && <span className="text-emerald-600">{t("product.verified")}</span>}
          </div>
        )}
        <Link
          to={`/products/${product.slug}`}
          className="mt-0.5 font-medium text-slate-800 leading-snug line-clamp-2 hover:text-brand-600 transition-colors"
        >
          {product.name}
        </Link>

        <div className="mt-auto pt-3 flex items-end justify-between gap-2">
          <div>
            <div className="text-lg font-semibold text-slate-900">{product.price.formatted}</div>
            <div className="text-[11px] text-slate-400">
              {product.stock > 0 ? t("product.inStock", { count: product.stock }) : t("product.outOfStock")}
              {product.min_order_qty > 1 && ` · ${t("product.moq", { count: product.min_order_qty })}`}
            </div>
          </div>
          <button
            disabled={product.stock <= 0}
            onClick={() => addToCart(product, product.min_order_qty)}
            className="btn-primary !px-3 !py-2 shrink-0"
            title={t("product.addToCart")}
          >
            {t("product.add")}
          </button>
        </div>
      </div>
    </div>
  );
}
