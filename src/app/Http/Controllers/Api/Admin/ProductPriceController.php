<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ProductPriceRequest;
use App\Http\Requests\Admin\ProductPriceSyncRequest;
use App\Models\Product;
use App\Models\ProductPrice;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Http\JsonResponse;
use Illuminate\Support\Arr;
use Illuminate\Support\Facades\DB;

class ProductPriceController extends Controller
{
    public function index(Product $product): JsonResponse
    {
        return response()->json($this->pricesOf($product));
    }

    public function store(ProductPriceRequest $request, Product $product): JsonResponse
    {
        $price = $product->prices()->create($request->validated());

        return response()->json($price, 201);
    }

    public function update(ProductPriceRequest $request, Product $product, ProductPrice $price): JsonResponse
    {
        $this->ensureBelongs($product, $price);
        $price->update($request->validated());

        return response()->json($price->fresh());
    }

    public function destroy(Product $product, ProductPrice $price): JsonResponse
    {
        $this->ensureBelongs($product, $price);
        $price->delete();

        return response()->json(['message' => 'Price deleted.']);
    }

    /**
     * Replace the product's whole price grid. Rows are rewritten rather than
     * matched by id so that reordering tiers can never trip the
     * (product, currency, min_qty) unique index mid-update.
     */
    public function sync(ProductPriceSyncRequest $request, Product $product): JsonResponse
    {
        $rows = array_map(
            fn (array $row) => Arr::only($row, ['currency_code', 'min_qty', 'price']),
            $request->validated('prices')
        );

        DB::transaction(function () use ($product, $rows) {
            $product->prices()->delete();
            $product->prices()->createMany($rows);
        });

        return response()->json($this->pricesOf($product));
    }

    private function ensureBelongs(Product $product, ProductPrice $price): void
    {
        abort_unless($price->product_id === $product->id, 404);
    }

    private function pricesOf(Product $product): Collection
    {
        return $product->prices()->orderBy('currency_code')->orderBy('min_qty')->get();
    }
}
