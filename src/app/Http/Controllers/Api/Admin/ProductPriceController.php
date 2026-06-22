<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\ProductPriceRequest;
use App\Models\Product;
use App\Models\ProductPrice;
use Illuminate\Http\JsonResponse;

class ProductPriceController extends Controller
{
    public function index(Product $product): JsonResponse
    {
        return response()->json(
            $product->prices()->orderBy('currency_code')->orderBy('min_qty')->get()
        );
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

    private function ensureBelongs(Product $product, ProductPrice $price): void
    {
        abort_unless($price->product_id === $product->id, 404);
    }
}
