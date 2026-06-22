<?php

namespace App\Enums;

use App\Enums\Concerns\HasValues;

enum UserRole: string
{
    use HasValues;

    case Customer = 'customer';
    case Manager = 'manager';
    case Admin = 'admin';
}
