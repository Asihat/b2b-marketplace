import { useEffect, useState } from "react";
import { Link, NavLink, Outlet, useNavigate } from "react-router-dom";
import { api, type Currency, type Language } from "../api";
import { useStore } from "../store";

export function Layout() {
  const { currency, setCurrency, locale, setLocale, user, logout, cartCount } = useStore();
  const [currencies, setCurrencies] = useState<Currency[]>([]);
  const [languages, setLanguages] = useState<Language[]>([]);
  const navigate = useNavigate();

  useEffect(() => {
    api.currencies().then(setCurrencies).catch(() => {});
    api.languages().then(setLanguages).catch(() => {});
  }, []);

  const selectClass =
    "appearance-none rounded-lg bg-white/70 ring-1 ring-slate-200 hover:ring-slate-300 px-2.5 py-1.5 text-sm text-slate-600 transition cursor-pointer focus:outline-none focus:ring-2 focus:ring-brand-500";

  const navLink = ({ isActive }: { isActive: boolean }) =>
    `text-sm transition-colors ${isActive ? "text-slate-900 font-medium" : "text-slate-500 hover:text-slate-900"}`;

  return (
    <div className="min-h-screen flex flex-col">
      <header className="glass sticky top-0 z-20 border-b border-slate-900/[0.06]">
        <div className="max-w-6xl mx-auto px-4 h-16 flex items-center gap-5">
          <Link to="/" className="flex items-center gap-2 shrink-0">
            <span className="grid place-items-center w-8 h-8 rounded-xl bg-gradient-to-br from-brand-500 to-violet-600 text-white text-sm shadow-sm shadow-brand-600/30">
              ◆
            </span>
            <span className="text-[15px] font-semibold tracking-tight text-slate-900">Marketplace</span>
          </Link>

          <nav className="hidden sm:flex items-center gap-5">
            <NavLink to="/" end className={navLink}>Catalog</NavLink>
            {user && <NavLink to="/orders" className={navLink}>Orders</NavLink>}
            {user?.role === "admin" && (
              <NavLink to="/admin" className={({ isActive }) =>
                `text-sm font-medium transition-colors ${isActive ? "text-brand-700" : "text-brand-600 hover:text-brand-700"}`
              }>
                Admin
              </NavLink>
            )}
          </nav>

          <div className="ml-auto flex items-center gap-2">
            <select value={locale} onChange={(e) => setLocale(e.target.value)} className={`${selectClass} hidden sm:block`} title="Language">
              {languages.map((l) => <option key={l.code} value={l.code}>{l.native_name}</option>)}
            </select>

            <select value={currency} onChange={(e) => setCurrency(e.target.value)} className={selectClass} title="Currency">
              {currencies.map((c) => <option key={c.code} value={c.code}>{c.code} {c.symbol}</option>)}
            </select>

            <Link
              to="/cart"
              className="relative grid place-items-center w-9 h-9 rounded-lg text-slate-600 hover:bg-slate-900/5 transition"
              title="Cart"
            >
              <span className="text-[17px]">🛍️</span>
              {cartCount > 0 && (
                <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] grid place-items-center bg-brand-600 text-white text-[10px] font-semibold rounded-full px-1 ring-2 ring-white">
                  {cartCount}
                </span>
              )}
            </Link>

            {user ? (
              <div className="flex items-center gap-2 pl-1">
                <div className="hidden md:flex items-center gap-2">
                  <span className="grid place-items-center w-8 h-8 rounded-full bg-slate-200 text-slate-600 text-xs font-semibold">
                    {user.name.slice(0, 1).toUpperCase()}
                  </span>
                  <div className="leading-tight">
                    <div className="text-xs font-medium text-slate-800">{user.name}</div>
                    <div className="text-[10px] uppercase tracking-wide text-slate-400">{user.type}</div>
                  </div>
                </div>
                <button onClick={() => { logout(); navigate("/"); }} className="text-sm text-slate-400 hover:text-rose-600 transition px-2">
                  Logout
                </button>
              </div>
            ) : (
              <Link to="/login" className="btn-primary !px-4 !py-2 ml-1">Sign in</Link>
            )}
          </div>
        </div>
      </header>

      <main className="flex-1 max-w-6xl w-full mx-auto px-4 py-8">
        <Outlet />
      </main>

      <footer className="border-t border-slate-900/[0.06] mt-8">
        <div className="max-w-6xl mx-auto px-4 py-6 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-slate-400">
          <span>© {new Date().getFullYear()} Marketplace — open-source B2B / B2C platform</span>
          <span>Laravel · Octane · React · Tailwind</span>
        </div>
      </footer>
    </div>
  );
}
