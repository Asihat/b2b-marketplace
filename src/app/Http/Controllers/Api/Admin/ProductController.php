<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ProductRequest;
use App\Models\Product;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

class ProductController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $products = Product::query()
            ->with(['category:id,name', 'company:id,name', 'images'])
            ->when($request->query('search'), fn ($q, $s) => $q->where(fn ($w) => $w
                ->where('name', 'ilike', "%{$s}%")
                ->orWhere('sku', 'ilike', "%{$s}%")
                ->orWhere('brand', 'ilike', "%{$s}%")))
            ->when($request->query('category_id'), fn ($q, $id) => $q->where('category_id', $id))
            ->latest()
            ->paginate(20);

        return response()->json($products);
    }

    public function store(ProductRequest $request): JsonResponse
    {
        $product = Product::create($this->attributes($request));
        $this->syncImages($product, $request->validated('images'));

        return response()->json($product->load(['category:id,name', 'images']), 201);
    }

    public function update(ProductRequest $request, Product $product): JsonResponse
    {
        $product->update($this->attributes($request));
        $this->syncImages($product, $request->validated('images'));

        return response()->json($product->fresh()->load(['category:id,name', 'images']));
    }

    public function destroy(Product $product): JsonResponse
    {
        $product->delete();

        return response()->json(['message' => 'Product deleted.']);
    }

    /** Persistable product columns (everything except the images list). */
    private function attributes(ProductRequest $request): array
    {
        $data = collect($request->validated())->except('images')->all();
        $data['slug'] ??= isset($data['sku']) ? Str::slug($data['sku']) : null;

        return array_filter($data, fn ($v) => $v !== null);
    }

    /** Replace a product's images from a list of URLs (first = primary). */
    private function syncImages(Product $product, ?array $urls): void
    {
        if ($urls === null) {
            return;
        }

        $product->images()->delete();
        foreach (array_values(array_filter($urls)) as $i => $url) {
            $product->images()->create([
                'url' => $url,
                'alt' => $product->name,
                'position' => $i + 1,
                'is_primary' => $i === 0,
            ]);
        }
    }
}
