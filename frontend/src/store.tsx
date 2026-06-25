import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api, setToken, type AppSettings, type Product, type User } from "./api";

// ---- Cart ----

export interface CartLine {
  product: Product;
  quantity: number;
}

interface StoreState {
  // settings
  currency: string;
  locale: string;
  settings: AppSettings;
  isB2bMode: boolean;
  refreshSettings: () => Promise<AppSettings>;
  setCurrency: (c: string) => void;
  setLocale: (l: string) => void;
  // auth
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  register: (payload: Record<string, unknown>) => Promise<void>;
  logout: () => void;
  // cart
  cart: CartLine[];
  addToCart: (product: Product, quantity: number) => void;
  removeFromCart: (productId: number) => void;
  clearCart: () => void;
  cartCount: number;
}

const StoreContext = createContext<StoreState | null>(null);
const DEFAULT_SETTINGS: AppSettings = { mode: "b2c", show_company_names: false };

export function StoreProvider({ children }: { children: ReactNode }) {
  const [currency, setCurrencyState] = useState(localStorage.getItem("currency") ?? "USD");
  const [locale, setLocaleState] = useState(localStorage.getItem("locale") ?? "en");
  const [settings, setSettings] = useState<AppSettings>(DEFAULT_SETTINGS);
  const [user, setUser] = useState<User | null>(null);
  const [cart, setCart] = useState<CartLine[]>([]);

  const setCurrency = (c: string) => {
    localStorage.setItem("currency", c);
    setCurrencyState(c);
  };
  const setLocale = (l: string) => {
    localStorage.setItem("locale", l);
    setLocaleState(l);
  };
  const refreshSettings = useCallback(async () => {
    const next = await api.settings();
    setSettings(next);
    return next;
  }, []);

  // Reflect the active UI locale on <html lang="…">.
  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  // Load public website configuration.
  useEffect(() => {
    refreshSettings().catch(() => setSettings(DEFAULT_SETTINGS));
  }, [refreshSettings]);

  // Restore session on load.
  useEffect(() => {
    if (localStorage.getItem("token")) {
      api.me().then(setUser).catch(() => setToken(null));
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const { user, token } = await api.login(email, password);
    setToken(token);
    setUser(user);
    if (user.currency) setCurrency(user.currency);
  }, []);

  const register = useCallback(async (payload: Record<string, unknown>) => {
    const { user, token } = await api.register(payload);
    setToken(token);
    setUser(user);
  }, []);

  const logout = useCallback(() => {
    setToken(null);
    setUser(null);
  }, []);

  const addToCart = useCallback((product: Product, quantity: number) => {
    setCart((prev) => {
      const existing = prev.find((l) => l.product.id === product.id);
      if (existing) {
        return prev.map((l) =>
          l.product.id === product.id ? { ...l, quantity: l.quantity + quantity } : l,
        );
      }
      return [...prev, { product, quantity }];
    });
  }, []);

  const removeFromCart = useCallback((productId: number) => {
    setCart((prev) => prev.filter((l) => l.product.id !== productId));
  }, []);

  const clearCart = useCallback(() => setCart([]), []);

  const value = useMemo<StoreState>(
    () => ({
      currency,
      locale,
      settings,
      isB2bMode: settings.mode === "b2b",
      refreshSettings,
      setCurrency,
      setLocale,
      user,
      login,
      register,
      logout,
      cart,
      addToCart,
      removeFromCart,
      clearCart,
      cartCount: cart.reduce((n, l) => n + l.quantity, 0),
    }),
    [currency, locale, settings, refreshSettings, user, cart, login, register, logout, addToCart, removeFromCart, clearCart],
  );

  return <StoreContext.Provider value={value}>{children}</StoreContext.Provider>;
}

export function useStore() {
  const ctx = useContext(StoreContext);
  if (!ctx) throw new Error("useStore must be used within StoreProvider");
  return ctx;
}
