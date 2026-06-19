<?php

namespace App\Http\Middleware;

use App\Models\Language;
use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Symfony\Component\HttpFoundation\Response;

/**
 * Resolves the active locale from (in order): ?lang= query param,
 * Accept-Language header, the authenticated user's preference, or the default.
 */
class SetLocale
{
    public function handle(Request $request, Closure $next): Response
    {
        $supported = Cache::remember('languages.codes', 3600, fn () =>
            Language::where('is_active', true)->pluck('code')->all()
        );

        $locale = $request->query('lang')
            ?? $request->user()?->locale
            ?? $request->getPreferredLanguage($supported ?: null)
            ?? config('app.locale');

        if (in_array($locale, $supported, true)) {
            app()->setLocale($locale);
        }

        return $next($request);
    }
}
