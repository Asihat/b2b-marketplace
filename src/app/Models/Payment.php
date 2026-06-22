<?php

namespace App\Models;

use App\Enums\PaymentStatus;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Payment extends Model
{
    protected $fillable = [
        'order_id', 'gateway', 'status', 'currency_code',
        'amount', 'reference', 'payload', 'paid_at',
    ];

    protected $casts = [
        'amount' => 'decimal:4',
        'payload' => 'array',
        'paid_at' => 'datetime',
        'status' => PaymentStatus::class,
    ];

    public function order(): BelongsTo
    {
        return $this->belongsTo(Order::class);
    }
}
