/** Formats a monetary amount with its currency code, e.g. "12.50 USD". */
export function formatMoney(amount: number | string, currency: string): string {
  return `${Number(amount).toFixed(2)} ${currency}`;
}

/** Thousands-separated integer, e.g. "12,480". */
export function formatNumber(value: number | string): string {
  return Number(value).toLocaleString("en-US");
}

/** Shortens large numbers for axis ticks and stat tiles, e.g. "12.5K", "4.2M". */
export function formatCompact(value: number): string {
  const abs = Math.abs(value);
  const [divisor, suffix] =
    abs >= 1e9 ? [1e9, "B"] : abs >= 1e6 ? [1e6, "M"] : abs >= 1e4 ? [1e3, "K"] : [1, ""];

  const scaled = value / divisor;
  // Keep one decimal only where it carries information (1.2K, not 1.0K).
  const digits = suffix && Math.abs(scaled) < 100 ? 1 : 0;

  return `${Number(scaled.toFixed(digits))}${suffix}`;
}

/** Compact money for tiles and axes, e.g. "$14.2K". */
export function formatCompactMoney(amount: number, symbol: string): string {
  return `${symbol}${formatCompact(amount)}`;
}

/** "2026-07-09" → "Jul 9". */
export function formatShortDate(isoDate: string): string {
  return new Date(`${isoDate}T00:00:00`).toLocaleDateString("en-US", { month: "short", day: "numeric" });
}
