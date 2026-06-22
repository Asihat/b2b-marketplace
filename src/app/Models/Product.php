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

    /** Sort by a whitelisted column/direction (guards against SQL injection). */
    public function scopeSorted(Builder $query, ?string $column, ?string $direction): Builder
    {
        $column = in_array($column, ['name', 'base_price', 'stock', 'created_at'], true) ? $column : 'name';
        $direction = strtolower((string) $direction) === 'desc' ? 'desc' : 'asc';

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
