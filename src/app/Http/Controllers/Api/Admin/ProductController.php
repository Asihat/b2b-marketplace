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
            ->withCount('analogs')
            ->search($request->query('search'))
            ->when($request->query('category_id'), fn ($q, $id) => $q->where('category_id', $id))
            ->when($request->filled('is_active'), fn ($q) => $q->where('is_active', $request->boolean('is_active')))
            ->when($request->filled('is_b2b_only'), fn ($q) => $q->where('is_b2b_only', $request->boolean('is_b2b_only')))
            ->when($request->query('exclude_id'), fn ($q, $id) => $q->whereKeyNot($id))
            ->latest()
            ->paginate(min(100, (int) $request->query('per_page', 20)));

        return response()->json($products);
    }

    public function show(Product $product): JsonResponse
    {
        return response()->json($this->withRelations($product));
    }

    public function store(ProductRequest $request): JsonResponse
    {
        $product = Product::create($this->attributes($request));
        $this->syncImages($product, $request->validated('images'));

        return response()->json($this->withRelations($product), 201);
    }

    public function update(ProductRequest $request, Product $product): JsonResponse
    {
        $product->update($this->attributes($request));
        $this->syncImages($product, $request->validated('images'));

        return response()->json($this->withRelations($product->fresh()));
    }

    public function destroy(Product $product): JsonResponse
    {
        $product->delete();

        return response()->json(['message' => 'Product deleted.']);
    }

    private function withRelations(Product $product): Product
    {
        return $product->load(['category:id,name', 'company:id,name', 'images', 'prices'])
            ->loadCount('analogs');
    }

    /** Persistable product columns (everything except the images list). */
    private function attributes(ProductRequest $request): array
    {
        $data = collect($request->validated())->except('images')->all();

        if (blank($data['slug'] ?? null)) {
            unset($data['slug']);

            if ($request->isMethod('post')) {
                $data['slug'] = Str::slug($request->validated('sku'));
            }
        }

        // Columns with a NOT NULL default: an omitted value keeps what is there.
        // Everything else (category, brand, description…) may be nulled out.
        foreach (['unit', 'stock', 'min_order_qty'] as $column) {
            if (array_key_exists($column, $data) && $data[$column] === null) {
                unset($data[$column]);
            }
        }

        return $data;
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
