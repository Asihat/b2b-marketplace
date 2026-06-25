import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import "./index.css";
import { StoreProvider } from "./store";
import { Layout } from "./components/Layout";
import { Catalog } from "./pages/Catalog";
import { ProductDetail } from "./pages/ProductDetail";
import { Login } from "./pages/Login";
import { Cart } from "./pages/Cart";
import { Orders } from "./pages/Orders";
import { AdminLayout } from "./components/AdminLayout";
import { Dashboard } from "./pages/admin/Dashboard";
import { Orders as AdminOrders } from "./pages/admin/Orders";
import { Products as AdminProducts } from "./pages/admin/Products";
import { Categories as AdminCategories } from "./pages/admin/Categories";
import { Users as AdminUsers } from "./pages/admin/Users";
import { Companies as AdminCompanies } from "./pages/admin/Companies";
import { Currencies as AdminCurrencies } from "./pages/admin/Currencies";
import { Settings as AdminSettings } from "./pages/admin/Settings";

const router = createBrowserRouter([
  {
    element: <Layout />,
    children: [
      { path: "/", element: <Catalog /> },
      { path: "/products/:slug", element: <ProductDetail /> },
      { path: "/login", element: <Login /> },
      { path: "/cart", element: <Cart /> },
      { path: "/orders", element: <Orders /> },
    ],
  },
  {
    path: "/admin",
    element: <AdminLayout />,
    children: [
      { index: true, element: <Dashboard /> },
      { path: "orders", element: <AdminOrders /> },
      { path: "products", element: <AdminProducts /> },
      { path: "categories", element: <AdminCategories /> },
      { path: "users", element: <AdminUsers /> },
      { path: "companies", element: <AdminCompanies /> },
      { path: "currencies", element: <AdminCurrencies /> },
      { path: "settings", element: <AdminSettings /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <StoreProvider>
      <RouterProvider router={router} />
    </StoreProvider>
  </StrictMode>,
);
