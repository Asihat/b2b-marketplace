/** Formats a monetary amount with its currency code, e.g. "12.50 USD". */
export function formatMoney(amount: number | string, currency: string): string {
  return `${Number(amount).toFixed(2)} ${currency}`;
}
