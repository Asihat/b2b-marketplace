<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Company extends Model
{
    protected $fillable = [
        'name', 'slug', 'tax_number', 'email', 'phone', 'country',
        'address', 'default_currency', 'default_locale', 'is_verified',
    ];

    protected $casts = [
        'is_verified' => 'boolean',
    ];

    public function users(): HasMany
    {
        return $this->hasMany(User::class);
    }

    public function products(): HasMany
    {
        return $this->hasMany(Product::class);
    }

    public function orders(): HasMany
    {
        return $this->hasMany(Order::class);
    }

    public function scopeSearch(Builder $query, ?string $term): Builder
    {
        return blank($term) ? $query : $query->where('name', 'ilike', "%{$term}%");
    }
}
