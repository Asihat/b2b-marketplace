<?php

namespace App\Enums\Concerns;

/**
 * Convenience helpers for string-backed enums (validation lists, etc.).
 */
trait HasValues
{
    /** @return array<int, string> */
    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
