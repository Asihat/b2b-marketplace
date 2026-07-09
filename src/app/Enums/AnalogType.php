<?php

namespace App\Enums;

use App\Enums\Concerns\HasValues;

/**
 * How an analog relates to the product it is linked from. Links are symmetric,
 * so both sides of a pair carry the same type.
 */
enum AnalogType: string
{
    use HasValues;

    case Equivalent = 'equivalent';
    case Substitute = 'substitute';
    case Upgrade = 'upgrade';
}
