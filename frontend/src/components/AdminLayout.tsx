import { NavLink, Outlet, Link, Navigate } from "react-router-dom";
import { useStore } from "../store";

const NAV: { to: string; label: string; icon: string; end?: boolean }[] = [
  { to: "/admin", label: "Dashboard", icon: "▦", end: true },
  { to: "/admin/orders", label: "Orders", icon: "🧾" },
  { to: "/admin/products", label: "Products", icon: "📦" },
  { to: "/admin/categories", label: "Categories", icon: "🗂" },
  { to: "/admin/users", label: "Users", icon: "👤" },
  { to: "/admin/companies", label: "Companies", icon: "🏢" },
  { to: "/admin/currencies", label: "Currencies", icon: "💱" },
];

export function AdminLayout() {
  const { user } = useStore();

  if (user === null && localStorage.getItem("token")) {
    return <div className="p-10 text-slate-400">Loading…</div>;
  }
  if (!user || user.role !== "admin") {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="min-h-screen flex bg-slate-50">
      <aside className="w-60 bg-slate-950 text-slate-300 flex flex-col fixed inset-y-0">
        <div className="px-5 h-16 flex items-center gap-2 border-b border-white/10">
          <span className="grid place-items-center w-8 h-8 rounded-xl bg-gradient-to-br from-brand-500 to-violet-600 text-white text-sm">◆</span>
          <span className="text-base font-semibold text-white tracking-tight">Admin</span>
        </div>
        <nav className="flex-1 py-3 px-3 space-y-0.5">
          {NAV.map((n) => (
            <NavLink
              key={n.to}
              to={n.to}
              end={n.end}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition ${
                  isActive ? "bg-brand-600 text-white shadow-sm shadow-brand-600/30" : "text-slate-400 hover:text-white hover:bg-white/5"
                }`
              }
            >
              <span className="text-base w-5 text-center">{n.icon}</span>
              {n.label}
            </NavLink>
          ))}
        </nav>
        <div className="px-5 py-4 border-t border-white/10 text-xs">
          <div className="flex items-center gap-2 mb-3">
            <span className="grid place-items-center w-7 h-7 rounded-full bg-white/10 text-white font-semibold">
              {user.name.slice(0, 1).toUpperCase()}
            </span>
            <span className="text-slate-300 truncate">{user.name}</span>
          </div>
          <Link to="/" className="text-brand-300 hover:text-brand-200 transition">← Back to storefront</Link>
        </div>
      </aside>

      <main className="flex-1 ml-60 p-8 overflow-auto">
        <Outlet />
      </main>
    </div>
  );
}
