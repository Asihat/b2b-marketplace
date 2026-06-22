<?php

namespace App\Models;

use App\Enums\AccountType;
use App\Enums\OrderStatus;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasOne;

class Order extends Model
{
    protected $fillable = [
        'number', 'user_id', 'company_id', 'type', 'status', 'currency_code',
        'subtotal', 'tax_total', 'grand_total', 'notes',
        'contact_name', 'contact_email', 'contact_phone',
        'shipping_address', 'shipping_city', 'shipping_postal_code', 'shipping_country',
    ];

    protected $casts = [
        'subtotal' => 'decimal:4',
        'tax_total' => 'decimal:4',
        'grand_total' => 'decimal:4',
        'status' => OrderStatus::class,
        'type' => AccountType::class,
    ];

    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }

    public function company(): BelongsTo
    {
        return $this->belongsTo(Company::class);
    }

    public function items(): HasMany
    {
        return $this->hasMany(OrderItem::class);
    }

    public function payment(): HasOne
    {
        return $this->hasOne(Payment::class)->latestOfMany();
    }

    public function payments(): HasMany
    {
        return $this->hasMany(Payment::class);
    }
}
