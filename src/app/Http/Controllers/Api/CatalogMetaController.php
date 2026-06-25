<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Category;
use App\Models\Currency;
use App\Models\Language;
use App\Services\MarketplaceSettings;
use Illuminate\Http\JsonResponse;

/**
 * Public storefront metadata: categories, currencies and languages.
 */
class CatalogMetaController extends Controller
{
    public function settings(): JsonResponse
    {
        return response()->json(app(MarketplaceSettings::class)->publicSettings());
    }

    public function categories(): JsonResponse
    {
        $categories = Category::where('is_active', true)
            ->orderBy('position')
            ->get()
            ->map(fn (Category $c) => [
                'id' => $c->id,
                'parent_id' => $c->parent_id,
                'slug' => $c->slug,
                'name' => $c->translatedName(),
            ]);

        return response()->json($categories);
    }

    public function currencies(): JsonResponse
    {
        return response()->json(
            Currency::where('is_active', true)->get(['code', 'name', 'symbol', 'exchange_rate', 'is_base'])
        );
    }

    public function languages(): JsonResponse
    {
        return response()->json(
            Language::where('is_active', true)->get(['code', 'name', 'native_name', 'is_default'])
        );
    }
}
