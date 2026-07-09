<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Product extends Model
{
    protected $fillable = [
        'category_id', 'company_id', 'sku', 'slug', 'name', 'description',
        'brand', 'unit', 'base_price', 'stock', 'min_order_qty',
        'is_b2b_only', 'is_active',
    ];

    protected $casts = [
        'base_price' => 'decimal:4',
        'is_b2b_only' => 'boolean',
        'is_active' => 'boolean',
    ];

    public function category(): BelongsTo
    {
        return $this->belongsTo(Category::class);
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function translations(): HasMany
    {
        return $this->hasMany(ProductTranslation::class);
    }

    public function images(): HasMany
    {
        return $this->hasMany(ProductImage::class)->orderBy('position');
    }

    /** URL of the primary image (falls back to the first, then null). */
    public function primaryImageUrl(): ?string
    {
        $images = $this->relationLoaded('images') ? $this->images : $this->images()->get();

        return ($images->firstWhere('is_primary', true) ?? $images->first())?->url;
    }

    public function prices(): HasMany
    {
        return $this->hasMany(ProductPrice::class);
    }

    public function orderItems(): HasMany
    {
        return $this->hasMany(OrderItem::class);
    }

    /** Products that are analogs of this product. */
    public function analogs(): BelongsToMany
    {
        return $this->belongsToMany(Product::class, 'product_analogs', 'product_id', 'analog_id')
            ->withPivot(['type', 'note'])
            ->withTimestamps();
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where('is_active', true);
    }

    /** Full-text-ish search across the human-facing identifiers. No-op when blank. */
    public function scopeSearch(Builder $query, ?string $term): Builder
    {
        if (blank($term)) {
            return $query;
        }

        return $query->where(fn (Builder $w) => $w
            ->where('name', 'ilike', "%{$term}%")
            ->orWhere('sku', 'ilike', "%{$term}%")
            ->orWhere('brand', 'ilike', "%{$term}%"));
    }

    /**
     * The price a buyer in a given currency actually pays, as SQL — used to sort
     * and filter by price. Mirrors CurrencyService::priceFor(): an override for
     * the requested currency wins as-is, otherwise a base-currency tier or the
     * base price is converted at the requested currency's exchange rate.
     */
    public static function effectivePriceSql(): string
    {
        return <<<'SQL'
COALESCE(
    (
        SELECT pp.price
        FROM product_prices pp
        WHERE pp.product_id = products.id
          AND pp.currency_code = ?
          AND pp.min_qty <= ?
        ORDER BY pp.min_qty DESC
        LIMIT 1
    ),
    (
        SELECT bp.price * CAST(? AS DECIMAL(18, 8))
        FROM product_prices bp
        WHERE bp.product_id = products.id
          AND bp.currency_code = ?
          AND bp.min_qty <= ?
        ORDER BY bp.min_qty DESC
        LIMIT 1
    ),
    products.base_price * CAST(? AS DECIMAL(18, 8))
)
SQL;
    }

    /** @return array<int, string|int|float> */
    public static function effectivePriceBindings(string $currency, int $qty): array
    {
        $currencies = app(\App\Services\CurrencyService::class);
        $currency = strtoupper($currency);
        $rate = $currencies->rate($currency);

        return [$currency, $qty, $rate, $currencies->base()['code'], $qty, $rate];
    }

    public function scopeSorted(Builder $query, ?string $column, ?string $direction, ?string $currency = null, int $qty = 1): Builder
    {
        if ($column === 'popular') {
            return $query
                ->withSum('orderItems as sold_qty', 'quantity')
                ->orderByDesc('sold_qty')
                ->orderBy('name');
        }

        $column = in_array($column, ['name', 'base_price', 'stock', 'created_at'], true) ? $column : 'name';
        $direction = strtolower((string) $direction) === 'desc' ? 'desc' : 'asc';

        if ($column === 'base_price' && $currency !== null) {
            return $query
                ->orderByRaw(self::effectivePriceSql().' '.$direction, self::effectivePriceBindings($currency, $qty))
                ->orderBy('name');
        }

        return $query->orderBy($column, $direction);
    }

    /** Hide B2B-only products from B2C (guest / b2c) buyers. */
    public function scopeVisibleTo(Builder $query, ?User $user): Builder
    {
        if ($user && $user->isB2b()) {
            return $query;
        }

        return $query->where('is_b2b_only', false);
    }

    public function translatedName(?string $locale = null): string
    {
        $locale ??= app()->getLocale();
        $t = $this->translations->firstWhere('locale', $locale);

        return $t?->name ?? $this->name;
    }
}
