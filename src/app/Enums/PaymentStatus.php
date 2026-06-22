<?php

namespace App\Enums;

use App\Enums\Concerns\HasValues;

enum PaymentStatus: string
{
    use HasValues;

    case Pending = 'pending';
    case Completed = 'completed';
    case Failed = 'failed';
    case Refunded = 'refunded';
}
