<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ProductAnalogRequest;
use App\Models\Product;
use Illuminate\Http\JsonResponse;
use Illuminate\Support\Facades\DB;

/**
 * Analog / cross-reference links between products. A link is stored in both
 * directions with the same type and note, so an analog of A is always an
 * analog of B. Every endpoint returns the product's full analog list.
 */
class ProductAnalogController extends Controller
{
    public function index(Product $product): JsonResponse
    {
        return response()->json($this->analogsOf($product));
    }

    public function store(ProductAnalogRequest $request, Product $product): JsonResponse
    {
        $analog = Product::findOrFail($request->validated('analog_id'));
        $this->link($product, $analog, $request->validated('type'), $request->validated('note'));

        return response()->json($this->analogsOf($product), 201);
    }

    public function update(ProductAnalogRequest $request, Product $product, Product $analog): JsonResponse
    {
        $this->ensureLinked($product, $analog);
        $this->link($product, $analog, $request->validated('type'), $request->validated('note'));

        return response()->json($this->analogsOf($product));
    }

    public function destroy(Product $product, Product $analog): JsonResponse
    {
        $this->ensureLinked($product, $analog);

        DB::transaction(function () use ($product, $analog) {
            $product->analogs()->detach($analog->id);
            $analog->analogs()->detach($product->id);
        });

        return response()->json($this->analogsOf($product));
    }

    /** Create or update the pair in both directions. */
    private function link(Product $product, Product $analog, string $type, ?string $note): void
    {
        $pivot = ['type' => $type, 'note' => $note];

        DB::transaction(function () use ($product, $analog, $pivot) {
            $product->analogs()->syncWithoutDetaching([$analog->id => $pivot]);
            $analog->analogs()->syncWithoutDetaching([$product->id => $pivot]);
        });
    }

    private function ensureLinked(Product $product, Product $analog): void
    {
        abort_unless($product->analogs()->whereKey($analog->id)->exists(), 404);
    }

    /** @return array<int, array<string, mixed>> */
    private function analogsOf(Product $product): array
    {
        return $product->analogs()
            ->with('images')
            ->orderBy('name')
            ->get()
            ->map(fn (Product $analog) => [
                'id' => $analog->id,
                'sku' => $analog->sku,
                'slug' => $analog->slug,
                'name' => $analog->name,
                'brand' => $analog->brand,
                'stock' => $analog->stock,
                'is_active' => $analog->is_active,
                'image' => $analog->primaryImageUrl(),
                'type' => $analog->pivot->type,
                'note' => $analog->pivot->note,
            ])
            ->all();
    }
}
