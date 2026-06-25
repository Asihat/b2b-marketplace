<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Http\Resources\AnalogResource;
use App\Http\Resources\ProductResource;
use App\Models\Product;
use App\Services\MarketplaceSettings;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;

class ProductController extends Controller
{
    /**
     * Paginated, filterable catalog. B2B-only items are hidden from B2C buyers.
     */
    public function index(Request $request): AnonymousResourceCollection
    {
        $currency = strtoupper((string) $request->query('currency', $request->user()?->currency ?? 'USD'));
        $qty = max(1, (int) $request->query('qty', 1));
        $priceSql = Product::effectivePriceSql();
        $priceBindings = Product::effectivePriceBindings($currency, $qty);
        $showCompanyNames = app(MarketplaceSettings::class)->showCompanyNames();
        $relations = $showCompanyNames
            ? ['translations', 'images', 'company:id,name,slug,is_verified']
            : ['translations', 'images'];

        $products = Product::query()
            ->active()
            ->visibleTo($request->user())
            ->with($relations)
            ->search($request->query('search'))
            ->when($request->query('category_id'), fn ($q, $id) => $q->where('category_id', $id))
            ->when($request->query('brand'), fn ($q, $brand) => $q->where('brand', $brand))
            ->when($request->query('company_id'), fn ($q, $id) => $q->where('company_id', $id))
            ->when($request->filled('min_price'), fn ($q) => $q->whereRaw($priceSql.' >= ?', [...$priceBindings, max(0, (float) $request->query('min_price'))]))
            ->when($request->filled('max_price'), fn ($q) => $q->whereRaw($priceSql.' <= ?', [...$priceBindings, max(0, (float) $request->query('max_price'))]))
            ->when($request->boolean('in_stock'), fn ($q) => $q->where('stock', '>', 0))
            ->sorted($request->query('sort'), $request->query('direction'), $currency, $qty)
            ->paginate(min(100, (int) $request->query('per_page', 20)));

        return ProductResource::collection($products);
    }

    public function show(Request $request, string $idOrSlug): ProductResource
    {
        $product = Product::query()
            ->active()
            ->visibleTo($request->user())
            ->with(app(MarketplaceSettings::class)->showCompanyNames()
                ? ['translations', 'prices', 'images', 'company:id,name,slug,is_verified', 'analogs.images']
                : ['translations', 'prices', 'images', 'analogs.images'])
            ->where(function ($q) use ($idOrSlug) {
                $q->where('slug', $idOrSlug);
                if (ctype_digit((string) $idOrSlug)) {
                    $q->orWhere('id', (int) $idOrSlug);
                }
            })
            ->firstOrFail();

        return new ProductResource($product);
    }

    /** Analogs / interchangeable cross-references for a product. */
    public function analogs(Request $request, Product $product): AnonymousResourceCollection
    {
        $analogs = $product->analogs()
            ->active()
            ->visibleTo($request->user())
            ->with('images')
            ->get();

        return AnalogResource::collection($analogs);
    }
}
