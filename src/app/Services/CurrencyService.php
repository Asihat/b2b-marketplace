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

    /** How many units of $currency one unit of the base currency buys. */
    public function rate(string $currency): float
    {
        return $this->active()[strtoupper($currency)]['exchange_rate'] ?? 1.0;
    }

    /** Convert an amount held in the base currency into $toCurrency. */
    public function convert(float $amount, string $toCurrency): float
    {
        return round($amount * $this->rate($toCurrency), 2);
    }

    /**
     * Resolve the unit price of a product for a given quantity.
     *
     * Order of preference: an explicit override the admin entered for the
     * requested currency (used verbatim — it is already in that currency), then
     * a B2B volume tier in the base currency, then the product base price. The
     * last two live in the base currency, so they are converted at the
     * requested currency's exchange rate.
     */
    public function priceFor(Product $product, string $currency, int $qty = 1): float
    {
        $currency = strtoupper($currency);

        // 1) Explicit price the admin set for this exact currency + quantity tier.
        $override = ProductPrice::where('product_id', $product->id)
            ->where('currency_code', $currency)
            ->where('min_qty', '<=', $qty)
            ->orderByDesc('min_qty')
            ->first();

        if ($override) {
            return round((float) $override->price, 2);
        }

        // 2) B2B volume tier defined in the base currency (so tiers still apply
        //    regardless of the currency the buyer is browsing in).
        $baseCode = $this->base()['code'];
        if ($currency !== $baseCode) {
            $baseTier = ProductPrice::where('product_id', $product->id)
                ->where('currency_code', $baseCode)
                ->where('min_qty', '<=', $qty)
                ->orderByDesc('min_qty')
                ->first();

            if ($baseTier) {
                return $this->convert((float) $baseTier->price, $currency);
            }
        }

        // 3) The admin-entered base price.
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
