<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Http\Resources\AnalogResource;
use App\Http\Resources\ProductResource;
use App\Models\Product;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;

class ProductController extends Controller
{
    /**
     * Paginated, filterable catalog. B2B-only items are hidden from B2C buyers.
     */
    public function index(Request $request): AnonymousResourceCollection
    {
        $products = Product::query()
            ->active()
            ->visibleTo($request->user())
            ->with(['translations', 'images'])
            ->search($request->query('search'))
            ->when($request->query('category_id'), fn ($q, $id) => $q->where('category_id', $id))
            ->when($request->query('brand'), fn ($q, $brand) => $q->where('brand', $brand))
            ->when($request->query('company_id'), fn ($q, $id) => $q->where('company_id', $id))
            ->when($request->boolean('in_stock'), fn ($q) => $q->where('stock', '>', 0))
            ->sorted($request->query('sort'), $request->query('direction'))
            ->paginate(min(100, (int) $request->query('per_page', 20)));

        return ProductResource::collection($products);
    }

    public function show(Request $request, string $idOrSlug): ProductResource
    {
        $product = Product::query()
            ->active()
            ->visibleTo($request->user())
            ->with(['translations', 'prices', 'images', 'analogs.images'])
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
