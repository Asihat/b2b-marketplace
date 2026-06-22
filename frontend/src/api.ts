const BASE = import.meta.env.VITE_API_URL ?? "http://localhost:8080/api";

// ---- Types mirroring the Laravel API resources ----

export interface Price {
  currency: string;
  amount: number;
  formatted: string;
}

export interface Product {
  id: number;
  sku: string;
  slug: string;
  name: string;
  description: string | null;
  brand: string | null;
  unit: string;
  stock: number;
  min_order_qty: number;
  is_b2b_only: boolean;
  category_id: number | null;
  company_id: number | null;
  image: string | null;
  images?: { url: string; alt: string | null; is_primary: boolean }[];
  price: Price;
  analogs?: Analog[];
}

export interface Analog {
  id: number;
  sku: string;
  slug: string;
  name: string;
  brand: string | null;
  stock: number;
  image: string | null;
  relation?: { type: string; note: string | null };
  price: Price;
}

export interface Category {
  id: number;
  parent_id: number | null;
  slug: string;
  name: string;
}

export interface Currency {
  code: string;
  name: string;
  symbol: string;
  exchange_rate: string;
  is_base: boolean;
}

export interface Language {
  code: string;
  name: string;
  native_name: string;
  is_default: boolean;
}

export interface User {
  id: number;
  name: string;
  email: string;
  type: "b2b" | "b2c";
  role: string;
  currency: string;
  locale: string;
  company?: { id: number; name: string } | null;
}

export interface OrderItem {
  id: number;
  name: string;
  sku: string;
  quantity: number;
  unit_price: string;
  line_total: string;
}

export interface Order {
  id: number;
  number: string;
  type: string;
  status: string;
  currency_code: string;
  subtotal: string;
  tax_total: string;
  grand_total: string;
  contact_name: string | null;
  contact_email: string | null;
  contact_phone: string | null;
  shipping_address: string | null;
  shipping_city: string | null;
  shipping_postal_code: string | null;
  shipping_country: string | null;
  notes: string | null;
  items?: OrderItem[];
  payment?: Payment | null;
}

export interface Payment {
  id: number;
  gateway: string;
  status: string;
  amount: string;
  reference: string | null;
}

export interface Paginated<T> {
  data: T[];
  meta: { current_page: number; last_page: number; total: number };
}

// Raw Laravel paginator (admin endpoints) — pagination keys at top level.
export interface Page<T> {
  data: T[];
  current_page: number;
  last_page: number;
  total: number;
}

export interface AdminUser extends User {
  is_active: boolean;
  phone: string | null;
  company_id: number | null;
}

export interface AdminProduct {
  id: number;
  sku: string;
  slug: string;
  name: string;
  brand: string | null;
  base_price: string;
  stock: number;
  min_order_qty: number;
  is_b2b_only: boolean;
  is_active: boolean;
  category_id: number | null;
  category?: { id: number; name: string } | null;
  images?: { id: number; url: string }[];
}

export interface AdminCategory {
  id: number;
  name: string;
  slug: string;
  parent_id: number | null;
  position: number;
  is_active: boolean;
  products_count?: number;
}

export interface AdminCompany {
  id: number;
  name: string;
  tax_number: string | null;
  country: string | null;
  is_verified: boolean;
  users_count?: number;
  products_count?: number;
  orders_count?: number;
}

export interface ProductPriceRow {
  id: number;
  product_id: number;
  currency_code: string;
  min_qty: number;
  price: string;
}

export interface Dashboard {
  stats: {
    users: number;
    b2b_users: number;
    companies: number;
    products: number;
    active_products: number;
    orders: number;
    pending_orders: number;
    revenue_by_currency: Record<string, string>;
  };
  recent_orders: (Order & { user?: { name: string; email: string } })[];
  low_stock: { id: number; sku: string; name: string; stock: number; min_order_qty: number }[];
}

// ---- Request helper ----

let authToken: string | null = localStorage.getItem("token");

export function setToken(token: string | null) {
  authToken = token;
  if (token) localStorage.setItem("token", token);
  else localStorage.removeItem("token");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (options.body) headers["Content-Type"] = "application/json";
  if (authToken) headers["Authorization"] = `Bearer ${authToken}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });
  const text = await res.text();
  const json = text ? JSON.parse(text) : null;

  if (!res.ok) {
    const message = json?.message ?? `Request failed (${res.status})`;
    throw new Error(message);
  }
  return json as T;
}

// ---- Query string builder ----

type Params = Record<string, string | number | boolean | undefined>;

function qs(params: Params): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== "");
  if (!entries.length) return "";
  return "?" + entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join("&");
}

// ---- API surface ----

export const api = {
  products: (params: Params) => request<Paginated<Product>>(`/products${qs(params)}`),
  product: (idOrSlug: string, params: Params) =>
    request<{ data: Product }>(`/products/${idOrSlug}${qs(params)}`).then((r) => r.data),
  analogs: (id: number, params: Params) =>
    request<{ data: Analog[] }>(`/products/${id}/analogs${qs(params)}`).then((r) => r.data),

  categories: () => request<Category[]>(`/categories`),
  currencies: () => request<Currency[]>(`/currencies`),
  languages: () => request<Language[]>(`/languages`),
  gateways: () => request<{ default: string; available: string[] }>(`/payment-gateways`),

  login: (email: string, password: string) =>
    request<{ user: User; token: string }>(`/auth/login`, {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  register: (payload: Record<string, unknown>) =>
    request<{ user: User; token: string }>(`/auth/register`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  me: () => request<User>(`/auth/me`),

  orders: () => request<Paginated<Order>>(`/orders`),
  createOrder: (payload: Record<string, unknown>) =>
    request<Order>(`/orders`, { method: "POST", body: JSON.stringify(payload) }),
  pay: (orderId: number, gateway: string) =>
    request<{ order: Order; payment: Payment }>(`/orders/${orderId}/pay`, {
      method: "POST",
      body: JSON.stringify({ gateway }),
    }),
};

// ---- Admin API (requires admin role) ----

export const adminApi = {
  dashboard: () => request<Dashboard>(`/admin/dashboard`),

  users: (params: Params) => request<Page<AdminUser>>(`/admin/users${qs(params)}`),
  saveUser: (id: number | null, payload: Record<string, unknown>) =>
    request<AdminUser>(`/admin/users${id ? `/${id}` : ""}`, {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    }),
  deleteUser: (id: number) => request<{ message: string }>(`/admin/users/${id}`, { method: "DELETE" }),

  orders: (params: Params) => request<Page<Order & { user?: { name: string; email: string } }>>(`/admin/orders${qs(params)}`),
  order: (id: number) => request<Order & { user?: { name: string; email: string } }>(`/admin/orders/${id}`),
  setOrderStatus: (id: number, status: string) =>
    request<Order>(`/admin/orders/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),

  products: (params: Params) => request<Page<AdminProduct>>(`/admin/products${qs(params)}`),
  saveProduct: (id: number | null, payload: Record<string, unknown>) =>
    request<AdminProduct>(`/admin/products${id ? `/${id}` : ""}`, {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    }),
  deleteProduct: (id: number) => request<{ message: string }>(`/admin/products/${id}`, { method: "DELETE" }),

  productPrices: (productId: number) =>
    request<ProductPriceRow[]>(`/admin/products/${productId}/prices`),
  saveProductPrice: (productId: number, id: number | null, payload: Record<string, unknown>) =>
    request<ProductPriceRow>(`/admin/products/${productId}/prices${id ? `/${id}` : ""}`, {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    }),
  deleteProductPrice: (productId: number, id: number) =>
    request<{ message: string }>(`/admin/products/${productId}/prices/${id}`, { method: "DELETE" }),

  categories: () => request<AdminCategory[]>(`/admin/categories`),
  saveCategory: (id: number | null, payload: Record<string, unknown>) =>
    request<AdminCategory>(`/admin/categories${id ? `/${id}` : ""}`, {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    }),
  deleteCategory: (id: number) => request<{ message: string }>(`/admin/categories/${id}`, { method: "DELETE" }),

  currencies: () => request<(Currency & { id: number; is_active: boolean })[]>(`/admin/currencies`),
  saveCurrency: (id: number | null, payload: Record<string, unknown>) =>
    request<Currency>(`/admin/currencies${id ? `/${id}` : ""}`, {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload),
    }),
  deleteCurrency: (id: number) => request<{ message: string }>(`/admin/currencies/${id}`, { method: "DELETE" }),

  companies: (params: Params) => request<Page<AdminCompany>>(`/admin/companies${qs(params)}`),
  updateCompany: (id: number, payload: Record<string, unknown>) =>
    request<AdminCompany>(`/admin/companies/${id}`, { method: "PUT", body: JSON.stringify(payload) }),
};
