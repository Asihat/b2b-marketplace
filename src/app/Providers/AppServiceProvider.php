<?php

namespace App\Providers;

use App\Payments\PaymentManager;
use App\Services\CurrencyService;
use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{
    public function register(): void
    {
        $this->app->singleton(CurrencyService::class);
        $this->app->singleton(PaymentManager::class);
    }

    public function boot(): void
    {
        //
    }
}
