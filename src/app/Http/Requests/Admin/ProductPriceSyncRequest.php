<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Validator;

/**
 * The full price grid of a product, submitted in one go. An empty list clears
 * every override and hands pricing back to the product's base price.
 */
class ProductPriceSyncRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    /** Currency codes are stored upper-cased. */
    protected function prepareForValidation(): void
    {
        $prices = array_map(function ($row) {
            if (is_array($row) && isset($row['currency_code'])) {
                $row['currency_code'] = strtoupper((string) $row['currency_code']);
            }

            return $row;
        }, (array) $this->input('prices', []));

        $this->merge(['prices' => $prices]);
    }

    public function rules(): array
    {
        return [
            'prices' => ['present', 'array', 'max:100'],
            'prices.*.currency_code' => ['required', 'string', 'size:3', 'exists:currencies,code'],
            'prices.*.min_qty' => ['required', 'integer', 'min:1'],
            'prices.*.price' => ['required', 'numeric', 'min:0'],
        ];
    }

    /** The (currency, min_qty) pair is unique per product — catch collisions before they hit the DB. */
    public function after(): array
    {
        return [function (Validator $validator) {
            $seen = [];

            foreach ($this->input('prices', []) as $i => $row) {
                $tier = ($row['currency_code'] ?? '').'|'.($row['min_qty'] ?? '');

                if (isset($seen[$tier])) {
                    $validator->errors()->add(
                        "prices.{$i}.min_qty",
                        "{$row['currency_code']} already has a tier starting at {$row['min_qty']}."
                    );
                }

                $seen[$tier] = true;
            }
        }];
    }
}
