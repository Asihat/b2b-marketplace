<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Product;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Illuminate\Validation\Rule;

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

    public function store(Request $request): JsonResponse
    {
        $data = $this->validateData($request);
        $images = $data['images'] ?? null;
        unset($data['images']);
        $data['slug'] = $data['slug'] ?? Str::slug($data['sku']);

        $product = Product::create($data);
        $this->syncImages($product, $images);

        return response()->json($product->load(['category:id,name', 'images']), 201);
    }

    public function update(Request $request, Product $product): JsonResponse
    {
        $data = $this->validateData($request, $product->id);
        $images = $data['images'] ?? null;
        unset($data['images']);

        $product->update($data);
        $this->syncImages($product, $images);

        return response()->json($product->fresh()->load(['category:id,name', 'images']));
    }

    /** Replace a product's images from a list of URLs (first = primary). */
    protected function syncImages(Product $product, ?array $urls): void
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

    public function destroy(Product $product): JsonResponse
    {
        $product->delete();

        return response()->json(['message' => 'Product deleted.']);
    }

    protected function validateData(Request $request, ?int $id = null): array
    {
        $creating = $id === null;

        return $request->validate([
            'sku' => [$creating ? 'required' : 'sometimes', 'string', Rule::unique('products', 'sku')->ignore($id)],
            'slug' => ['nullable', 'string', Rule::unique('products', 'slug')->ignore($id)],
            'name' => [$creating ? 'required' : 'sometimes', 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'brand' => ['nullable', 'string', 'max:255'],
            'unit' => ['nullable', 'string', 'max:50'],
            'category_id' => ['nullable', 'exists:categories,id'],
            'company_id' => ['nullable', 'exists:companies,id'],
            'base_price' => [$creating ? 'required' : 'sometimes', 'numeric', 'min:0'],
            'stock' => ['nullable', 'integer', 'min:0'],
            'min_order_qty' => ['nullable', 'integer', 'min:1'],
            'is_b2b_only' => ['boolean'],
            'is_active' => ['boolean'],
            'images' => ['nullable', 'array'],
            'images.*' => ['string', 'max:2048'],
        ]);
    }
}
