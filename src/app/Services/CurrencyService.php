<?php

namespace App\Services;

use App\Models\Currency;
use App\Models\Product;
use App\Models\ProductPrice;
use Illuminate\Support\Facades\Cache;

/**
 * Multi-currency pricing. Prices are stored in a base currency; other
 * currencies are derived from per-currency exchange rates, unless an explicit
 * ProductPrice override exists (used for fixed/round prices and B2B tiers).
 */
class CurrencyService
{
    /**
     * Active currencies as plain arrays keyed by code. Stored as arrays (not
     * Eloquent models) so the file cache can serialize/restore them safely.
     *
     * @return array<string, array{code:string,name:string,symbol:string,exchange_rate:float,is_base:bool}>
     */
    public function active(): array
    {
        return Cache::remember('currencies.active', 3600, function () {
            return Currency::where('is_active', true)
                ->get()
                ->keyBy('code')
                ->map(fn (Currency $c) => [
                    'code' => $c->code,
                    'name' => $c->name,
                    'symbol' => $c->symbol,
                    'exchange_rate' => (float) $c->exchange_rate,
                    'is_base' => (bool) $c->is_base,
                ])
                ->all();
        });
    }

    /** @return array{code:string,symbol:string,exchange_rate:float,is_base:bool} */
    public function base(): array
    {
        foreach ($this->active() as $currency) {
            if ($currency['is_base']) {
                return $currency;
            }
        }

        // Fallback so callers always get something usable.
        return ['code' => config('app.base_currency', 'USD'), 'symbol' => '$', 'exchange_rate' => 1.0, 'is_base' => true];
    }

    /** Convert an amount from the base currency into the target currency. */
    public function convert(float $amount, string $toCurrency): float
    {
        $currencies = $this->active();
        $rate = isset($currencies[$toCurrency]) ? (float) $currencies[$toCurrency]['exchange_rate'] : 1.0;

        return round($amount * $rate, 2);
    }

    /**
     * Resolve the unit price of a product in a currency for a given quantity,
     * honouring B2B volume tiers and explicit per-currency overrides.
     */
    public function priceFor(Product $product, string $currency, int $qty = 1): float
    {
        $override = ProductPrice::where('product_id', $product->id)
            ->where('currency_code', $currency)
            ->where('min_qty', '<=', $qty)
            ->orderByDesc('min_qty')
            ->first();

        if ($override) {
            return (float) $override->price;
        }

        return $this->convert((float) $product->base_price, $currency);
    }

    public function format(float $amount, string $currency): string
    {
        $symbol = $this->active()[$currency]['symbol'] ?? $currency.' ';

        return $symbol.number_format($amount, 2);
    }

    public function flush(): void
    {
        Cache::forget('currencies.active');
    }
}
