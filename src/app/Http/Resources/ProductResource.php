<?php

namespace App\Http\Resources;

use App\Services\CurrencyService;
use App\Services\MarketplaceSettings;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\JsonResource;

/**
 * @mixin \App\Models\Product
 */
class ProductResource extends JsonResource
{
    public function toArray(Request $request): array
    {
        $currency = $request->query('currency', $request->user()?->currency ?? 'USD');
        $qty = max(1, (int) $request->query('qty', 1));
        $currencyService = app(CurrencyService::class);
        $showCompanyNames = app(MarketplaceSettings::class)->showCompanyNames();

        return [
            'id' => $this->id,
            'sku' => $this->sku,
            'slug' => $this->slug,
            'name' => $this->translatedName(app()->getLocale()),
            'description' => $this->description,
            'brand' => $this->brand,
            'unit' => $this->unit,
            'stock' => $this->stock,
            'min_order_qty' => $this->min_order_qty,
            'is_b2b_only' => $this->is_b2b_only,
            'category_id' => $this->category_id,
            'company_id' => $this->company_id,
            'company' => $this->when($showCompanyNames, fn () => $this->whenLoaded('company', fn () => $this->company ? [
                'id' => $this->company->id,
                'name' => $this->company->name,
                'slug' => $this->company->slug,
                'is_verified' => $this->company->is_verified,
            ] : null)),
            'image' => $this->primaryImageUrl(),
            'images' => $this->whenLoaded('images', fn () => $this->images->map(fn ($img) => [
                'url' => $img->url,
                'alt' => $img->alt,
                'is_primary' => $img->is_primary,
            ])),
            'price' => [
                'currency' => $currency,
                'amount' => $currencyService->priceFor($this->resource, $currency, $qty),
                'formatted' => $currencyService->format(
                    $currencyService->priceFor($this->resource, $currency, $qty),
                    $currency
                ),
            ],
            'analogs' => AnalogResource::collection($this->whenLoaded('analogs')),
            'translations' => $this->whenLoaded('translations'),
        ];
    }
}
