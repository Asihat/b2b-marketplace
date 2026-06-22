<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class ProductPriceRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    /** Currency codes are stored upper-cased. */
    protected function prepareForValidation(): void
    {
        if ($this->has('currency_code')) {
            $this->merge(['currency_code' => strtoupper((string) $this->input('currency_code'))]);
        }
    }

    public function rules(): array
    {
        $productId = $this->route('product')?->id;
        $priceId = $this->route('price')?->id;

        return [
            // One price per product + currency + quantity tier.
            'currency_code' => [
                'required', 'string', 'size:3', 'exists:currencies,code',
                Rule::unique('product_prices', 'currency_code')
                    ->where('product_id', $productId)
                    ->where('min_qty', (int) $this->input('min_qty'))
                    ->ignore($priceId),
            ],
            'min_qty' => ['required', 'integer', 'min:1'],
            'price' => ['required', 'numeric', 'min:0'],
        ];
    }
}
