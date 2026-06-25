import { useStore } from "./store";

type Vars = Record<string, string | number>;

/**
 * UI translations for the storefront. Locale codes match the backend
 * `languages` table (en, ru, kk). Missing keys fall back to English.
 */
const dictionaries = {
  en: {
    nav: { catalog: "Catalog", orders: "Orders", admin: "Admin", signIn: "Sign in", logout: "Logout", cart: "Cart", language: "Language", currency: "Currency" },
    footer: { tagline: "Open-source B2B / B2C platform" },
    common: { loading: "Loading…", all: "All", total: "Total", back: "Back" },
    catalog: {
      heroBadge: "Wholesale & retail",
      heroTitle: "Everything your business needs, in one catalog.",
      heroSubtitle: "Multi-currency pricing, B2B volume tiers and verified suppliers — browse {count} products across every category.",
      searchPlaceholder: "Search products, SKU, brand…",
      minPrice: "Min price",
      maxPrice: "Max price",
      sortLabel: "Sort products",
      sortName: "Name",
      sortPriceAsc: "Price: low to high",
      sortPriceDesc: "Price: high to low",
      sortNewest: "Newest",
      sortPopular: "Popular",
      clearFilters: "Clear filters",
      empty: "No products match your search.",
      paginationLabel: "Catalog pagination",
      previousPage: "Previous page",
      nextPage: "Next page",
      pageSummary: "Page {page} of {pages} · {total} products",
    },
    product: {
      b2bOnly: "B2B only", soldOut: "Sold out",
      inStock: "{count} in stock", outOfStock: "Out of stock",
      moq: "MOQ {count}", minOrder: "min order {count}",
      add: "Add", addToCart: "Add to cart", added: "✓ Added to cart",
      businessOnly: "Business only", perUnit: "per {unit}",
      seller: "Seller",
      verified: "Verified",
      priceNote: "Price updates automatically for B2B volume tiers as quantity changes.",
      backToCatalog: "Back to catalog",
      analogs: "Analogs & cross-references", noAnalogs: "No analogs listed for this product.",
    },
    login: {
      welcomeBack: "Welcome back", createAccount: "Create your account",
      signinSub: "Sign in to order and track your purchases.",
      registerSub: "Join as a retail or business buyer.",
      login: "Login", register: "Register",
      fullName: "Full name", companyName: "Company name", email: "Email", password: "Password",
      signInBtn: "Sign in", createBtn: "Create account", pleaseWait: "Please wait…",
      demo: "Demo accounts — password: password",
    },
    cart: {
      title: "Cart", empty: "Your cart is empty.",
      browse: "Browse the catalog", viewOrders: "View your orders",
      contactDelivery: "Contact & delivery",
      contactName: "Contact name", emailUpdates: "Email (for order updates)",
      phone: "Phone", country: "Country (ISO-2)", shippingAddress: "Shipping address",
      city: "City", postal: "Postal code", notes: "Order notes",
      summary: "Summary", items: "Items", payment: "Payment method",
      checkout: "Checkout & pay", signinToCheckout: "Sign in to checkout", processing: "Processing…",
      addressRequired: "Please enter a shipping address.",
      orderPlaced: "Order {number} placed — {status} (payment {pstatus} via {gateway}).",
    },
    orders: {
      title: "Your orders", signin: "Sign in", signinToView: "to view your orders.",
      none: "No orders yet.", paidVia: "paid via {gateway}",
    },
    status: { pending: "Pending", paid: "Paid", processing: "Processing", shipped: "Shipped", completed: "Completed", cancelled: "Cancelled" },
  },

  ru: {
    nav: { catalog: "Каталог", orders: "Заказы", admin: "Админка", signIn: "Войти", logout: "Выйти", cart: "Корзина", language: "Язык", currency: "Валюта" },
    footer: { tagline: "Маркетплейс B2B / B2C с открытым кодом" },
    common: { loading: "Загрузка…", all: "Все", total: "Итого", back: "Назад" },
    catalog: {
      heroBadge: "Опт и розница",
      heroTitle: "Всё для вашего бизнеса в одном каталоге.",
      heroSubtitle: "Цены в разных валютах, оптовые уровни B2B и проверенные поставщики — {count} товаров во всех категориях.",
      searchPlaceholder: "Поиск товаров, артикула, бренда…",
      minPrice: "Цена от",
      maxPrice: "Цена до",
      sortLabel: "Сортировка товаров",
      sortName: "По названию",
      sortPriceAsc: "Цена: по возрастанию",
      sortPriceDesc: "Цена: по убыванию",
      sortNewest: "Новинки",
      sortPopular: "Популярные",
      clearFilters: "Сбросить фильтры",
      empty: "По вашему запросу ничего не найдено.",
      paginationLabel: "Пагинация каталога",
      previousPage: "Предыдущая страница",
      nextPage: "Следующая страница",
      pageSummary: "Страница {page} из {pages} · товаров: {total}",
    },
    product: {
      b2bOnly: "Только B2B", soldOut: "Нет в наличии",
      inStock: "{count} в наличии", outOfStock: "Нет в наличии",
      moq: "мин. {count}", minOrder: "мин. заказ {count}",
      add: "В корзину", addToCart: "В корзину", added: "✓ Добавлено",
      businessOnly: "Только для бизнеса", perUnit: "за {unit}",
      seller: "Продавец",
      verified: "Проверен",
      priceNote: "Цена автоматически обновляется по оптовым уровням B2B при изменении количества.",
      backToCatalog: "Назад в каталог",
      analogs: "Аналоги и кросс-ссылки", noAnalogs: "Аналоги для этого товара не указаны.",
    },
    login: {
      welcomeBack: "С возвращением", createAccount: "Создать аккаунт",
      signinSub: "Войдите, чтобы заказывать и отслеживать покупки.",
      registerSub: "Зарегистрируйтесь как розничный или бизнес-покупатель.",
      login: "Вход", register: "Регистрация",
      fullName: "Полное имя", companyName: "Название компании", email: "Эл. почта", password: "Пароль",
      signInBtn: "Войти", createBtn: "Создать аккаунт", pleaseWait: "Подождите…",
      demo: "Демо-аккаунты — пароль: password",
    },
    cart: {
      title: "Корзина", empty: "Ваша корзина пуста.",
      browse: "Перейти в каталог", viewOrders: "Посмотреть заказы",
      contactDelivery: "Контакты и доставка",
      contactName: "Контактное лицо", emailUpdates: "Эл. почта (для уведомлений)",
      phone: "Телефон", country: "Страна (ISO-2)", shippingAddress: "Адрес доставки",
      city: "Город", postal: "Почтовый индекс", notes: "Комментарий к заказу",
      summary: "Итого", items: "Товары", payment: "Способ оплаты",
      checkout: "Оформить и оплатить", signinToCheckout: "Войти для оформления", processing: "Обработка…",
      addressRequired: "Укажите адрес доставки.",
      orderPlaced: "Заказ {number} оформлен — {status} (оплата {pstatus} через {gateway}).",
    },
    orders: {
      title: "Ваши заказы", signin: "Войдите", signinToView: "чтобы посмотреть заказы.",
      none: "Заказов пока нет.", paidVia: "оплачено через {gateway}",
    },
    status: { pending: "В ожидании", paid: "Оплачен", processing: "В обработке", shipped: "Отправлен", completed: "Завершён", cancelled: "Отменён" },
  },

  kk: {
    nav: { catalog: "Каталог", orders: "Тапсырыстар", admin: "Әкімші", signIn: "Кіру", logout: "Шығу", cart: "Себет", language: "Тіл", currency: "Валюта" },
    footer: { tagline: "Ашық бастапқы кодты B2B / B2C маркетплейсі" },
    common: { loading: "Жүктелуде…", all: "Барлығы", total: "Барлығы", back: "Артқа" },
    catalog: {
      heroBadge: "Көтерме және бөлшек",
      heroTitle: "Бизнесіңізге қажеттің бәрі бір каталогта.",
      heroSubtitle: "Әртүрлі валютадағы бағалар, B2B көтерме деңгейлері және тексерілген жеткізушілер — барлық санаттарда {count} тауар.",
      searchPlaceholder: "Тауар, SKU, бренд іздеу…",
      minPrice: "Ең төмен баға",
      maxPrice: "Ең жоғары баға",
      sortLabel: "Тауарларды сұрыптау",
      sortName: "Атауы бойынша",
      sortPriceAsc: "Баға: төменнен жоғары",
      sortPriceDesc: "Баға: жоғарыдан төмен",
      sortNewest: "Жаңалары",
      sortPopular: "Танымал",
      clearFilters: "Сүзгілерді тазалау",
      empty: "Сұранысыңыз бойынша ештеңе табылмады.",
      paginationLabel: "Каталог беттері",
      previousPage: "Алдыңғы бет",
      nextPage: "Келесі бет",
      pageSummary: "{page}/{pages} бет · {total} тауар",
    },
    product: {
      b2bOnly: "Тек B2B", soldOut: "Сатылып кетті",
      inStock: "{count} қоймада бар", outOfStock: "Қоймада жоқ",
      moq: "ең аз {count}", minOrder: "ең аз тапсырыс {count}",
      add: "Себетке", addToCart: "Себетке қосу", added: "✓ Қосылды",
      businessOnly: "Тек бизнеске", perUnit: "{unit} үшін",
      seller: "Сатушы",
      verified: "Тексерілген",
      priceNote: "Баға B2B көтерме деңгейлеріне сай саны өзгергенде автоматты түрде жаңарады.",
      backToCatalog: "Каталогқа оралу",
      analogs: "Аналогтар және баламалар", noAnalogs: "Бұл тауарға аналогтар көрсетілмеген.",
    },
    login: {
      welcomeBack: "Қайта келдіңіз", createAccount: "Аккаунт жасау",
      signinSub: "Тапсырыс беру және сатып алуларды бақылау үшін кіріңіз.",
      registerSub: "Бөлшек немесе бизнес сатып алушы ретінде тіркеліңіз.",
      login: "Кіру", register: "Тіркелу",
      fullName: "Толық аты", companyName: "Компания атауы", email: "Эл. пошта", password: "Құпиясөз",
      signInBtn: "Кіру", createBtn: "Аккаунт жасау", pleaseWait: "Күте тұрыңыз…",
      demo: "Демо аккаунттар — құпиясөз: password",
    },
    cart: {
      title: "Себет", empty: "Себетіңіз бос.",
      browse: "Каталогты ашу", viewOrders: "Тапсырыстарды көру",
      contactDelivery: "Байланыс және жеткізу",
      contactName: "Байланыс тұлғасы", emailUpdates: "Эл. пошта (хабарламалар үшін)",
      phone: "Телефон", country: "Ел (ISO-2)", shippingAddress: "Жеткізу мекенжайы",
      city: "Қала", postal: "Пошта индексі", notes: "Тапсырысқа ескертпе",
      summary: "Қорытынды", items: "Тауарлар", payment: "Төлем әдісі",
      checkout: "Рәсімдеу және төлеу", signinToCheckout: "Рәсімдеу үшін кіру", processing: "Өңделуде…",
      addressRequired: "Жеткізу мекенжайын енгізіңіз.",
      orderPlaced: "Тапсырыс {number} рәсімделді — {status} (төлем {pstatus}, {gateway} арқылы).",
    },
    orders: {
      title: "Тапсырыстарыңыз", signin: "Кіріңіз", signinToView: "тапсырыстарды көру үшін.",
      none: "Әзірге тапсырыс жоқ.", paidVia: "{gateway} арқылы төленді",
    },
    status: { pending: "Күтуде", paid: "Төленген", processing: "Өңделуде", shipped: "Жіберілген", completed: "Аяқталған", cancelled: "Бас тартылған" },
  },
} as const;

export type Locale = keyof typeof dictionaries;
export const SUPPORTED_LOCALES = Object.keys(dictionaries) as Locale[];

function lookup(dict: unknown, path: string): unknown {
  return path.split(".").reduce<unknown>((o, k) => (o as Record<string, unknown>)?.[k], dict);
}

export function translate(locale: string, key: string, vars?: Vars): string {
  const dict = dictionaries[(locale as Locale)] ?? dictionaries.en;
  let value = lookup(dict, key);
  if (typeof value !== "string") value = lookup(dictionaries.en, key);
  if (typeof value !== "string") return key;

  let str = value;
  if (vars) {
    for (const [k, v] of Object.entries(vars)) {
      str = str.replaceAll(`{${k}}`, String(v));
    }
  }
  return str;
}

/** Hook returning a `t(key, vars?)` bound to the current UI locale. */
export function useT() {
  const { locale } = useStore();
  return (key: string, vars?: Vars) => translate(locale, key, vars);
}
