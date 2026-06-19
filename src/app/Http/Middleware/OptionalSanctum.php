<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Symfony\Component\HttpFoundation\Response;

/**
 * Resolves the Sanctum user when a bearer token is present, but lets guests
 * through. Used on public storefront routes so B2B buyers see B2B-only items
 * and their saved locale/currency while anonymous browsing still works.
 */
class OptionalSanctum
{
    public function handle(Request $request, Closure $next): Response
    {
        if ($request->bearerToken() && ($user = Auth::guard('sanctum')->user())) {
            $request->setUserResolver(fn () => $user);
        }

        return $next($request);
    }
}
