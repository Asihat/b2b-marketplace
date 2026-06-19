<?php

namespace App\Http\Resources;

use App\Services\CurrencyService;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

/**
 * A lightweight product reference used when listing analogs / cross-references.
 *
 * @mixin \App\Models\Product
 */
class AnalogResource extends JsonResource
{
    public function toArray(Request $request): array
    {
        $currency = $request->query('currency', $request->user()?->currency ?? 'USD');
        $currencyService = app(CurrencyService::class);

        return [
            'id' => $this->id,
            'sku' => $this->sku,
            'slug' => $this->slug,
            'name' => $this->translatedName(app()->getLocale()),
            'brand' => $this->brand,
            'stock' => $this->stock,
            'image' => $this->primaryImageUrl(),
            'relation' => $this->whenPivotLoaded('product_analogs', fn () => [
                'type' => $this->pivot->type,
                'note' => $this->pivot->note,
            ]),
            'price' => [
                'currency' => $currency,
                'amount' => $currencyService->priceFor($this->resource, $currency),
            ],
        ];
    }
}
