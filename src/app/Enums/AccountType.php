<?php

namespace App\Enums;

use App\Enums\Concerns\HasValues;

/**
 * Distinguishes retail (B2C) buyers from business (B2B) accounts. Shared by
 * users and orders.
 */
enum AccountType: string
{
    use HasValues;

    case B2c = 'b2c';
    case B2b = 'b2b';
}
