<?php

namespace App\Services;

use App\Models\AppSetting;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Storage;

class MarketplaceSettings
{
    private const CACHE_KEY = 'marketplace.settings';

    /** @return array{mode:string,show_company_names:bool,icon_url:?string,company_name:string,company_description:string} */
    public function publicSettings(): array
    {
        $mode = $this->mode();

        return [
            'mode' => $mode,
            'show_company_names' => $mode === 'b2b',
            'icon_url' => $this->iconUrl(),
            'company_name' => $this->companyName(),
            'company_description' => $this->companyDescription(),
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

    public function companyName(): string
    {
        return $this->all()['company_name'] ?? config('marketplace.company_name', 'Marketplace');
    }

    public function companyDescription(): string
    {
        return $this->all()['company_description'] ?? config('marketplace.company_description', '');
    }

    public function iconUrl(): ?string
    {
        $path = $this->all()['main_icon_path'] ?? null;

        return $path ? Storage::disk('public')->url($path) : null;
    }

    public function updateIcon(string $path): array
    {
        $oldPath = AppSetting::where('key', 'main_icon_path')->value('value');

        AppSetting::updateOrCreate(
            ['key' => 'main_icon_path'],
            ['value' => $path],
        );

        $this->flush();

        if ($oldPath && $oldPath !== $path && str_starts_with($oldPath, 'settings/icons/')) {
            Storage::disk('public')->delete($oldPath);
        }

        return $this->publicSettings();
    }

    public function removeIcon(): array
    {
        $path = AppSetting::where('key', 'main_icon_path')->value('value');

        AppSetting::where('key', 'main_icon_path')->delete();
        $this->flush();

        if ($path && str_starts_with($path, 'settings/icons/')) {
            Storage::disk('public')->delete($path);
        }

        return $this->publicSettings();
    }

    /** @param array{mode?:string,company_name?:string,company_description?:string|null} $settings */
    public function update(array $settings): array
    {
        $values = [
            'mode' => 'marketplace_mode',
            'company_name' => 'company_name',
            'company_description' => 'company_description',
        ];

        foreach ($values as $input => $key) {
            if (array_key_exists($input, $settings)) {
                AppSetting::updateOrCreate(
                    ['key' => $key],
                    ['value' => $settings[$input] ?? ''],
                );
            }
        }

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
