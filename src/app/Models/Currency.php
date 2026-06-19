<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class Currency extends Model
{
    protected $fillable = [
        'code', 'name', 'symbol', 'exchange_rate', 'is_base', 'is_active',
    ];

    protected $casts = [
        'exchange_rate' => 'decimal:8',
        'is_base' => 'boolean',
        'is_active' => 'boolean',
    ];

    public static function base(): ?self
    {
        return static::where('is_base', true)->first();
    }
}
