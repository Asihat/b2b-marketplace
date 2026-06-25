<?php

namespace App\Services;

use App\Models\AppSetting;
use Illuminate\Support\Facades\Cache;

class MarketplaceSettings
{
    private const CACHE_KEY = 'marketplace.settings';

    /** @return array{mode:string,show_company_names:bool} */
    public function publicSettings(): array
    {
        $mode = $this->mode();

        return [
            'mode' => $mode,
            'show_company_names' => $mode === 'b2b',
        ];
    }

    public function mode(): string
    {
        $mode = $this->all()['marketplace_mode'] ?? config('marketplace.mode', 'b2c');

        return in_array($mode, ['b2c', 'b2b'], true) ? $mode : 'b2c';
    }

    public function showCompanyNames(): bool
    {
        return $this->mode() === 'b2b';
    }

    /** @param array{mode:string} $settings */
    public function update(array $settings): array
    {
        $mode = in_array($settings['mode'], ['b2c', 'b2b'], true) ? $settings['mode'] : 'b2c';

        AppSetting::updateOrCreate(
            ['key' => 'marketplace_mode'],
            ['value' => $mode],
        );

        $this->flush();

        return $this->publicSettings();
    }

    public function flush(): void
    {
        Cache::forget(self::CACHE_KEY);
    }

    /** @return array<string, string|null> */
    private function all(): array
    {
        return Cache::rememberForever(self::CACHE_KEY, fn () => AppSetting::query()
            ->pluck('value', 'key')
            ->all());
    }
}
