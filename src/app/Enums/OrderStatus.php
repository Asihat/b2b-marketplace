<?php

namespace App\Enums;

use App\Enums\Concerns\HasValues;

enum OrderStatus: string
{
    use HasValues;

    case Pending = 'pending';
    case Paid = 'paid';
    case Processing = 'processing';
    case Shipped = 'shipped';
    case Completed = 'completed';
    case Cancelled = 'cancelled';

    /** Whether the order has been settled (no further payment required). */
    public function isSettled(): bool
    {
        return in_array($this, [self::Paid, self::Completed], true);
    }
}
