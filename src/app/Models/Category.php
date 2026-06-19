<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Category extends Model
{
    protected $fillable = [
        'parent_id', 'slug', 'name', 'name_translations', 'position', 'is_active',
    ];

    protected $casts = [
        'name_translations' => 'array',
        'is_active' => 'boolean',
    ];

    public function parent(): BelongsTo
    {
        return $this->belongsTo(Category::class, 'parent_id');
    }

    public function children(): HasMany
    {
        return $this->hasMany(Category::class, 'parent_id');
    }

    public function products(): HasMany
    {
        return $this->hasMany(Product::class);
    }

    /** Localized name with fallback to the default name. */
    public function translatedName(?string $locale = null): string
    {
        $locale ??= app()->getLocale();

        return $this->name_translations[$locale] ?? $this->name;
    }
}
