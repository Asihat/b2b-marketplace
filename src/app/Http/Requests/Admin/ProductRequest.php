<?php

namespace App\Http\Requests\Admin;

use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class ProductRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        $id = $this->route('product')?->id;
        $required = $this->isMethod('post') ? 'required' : 'sometimes';

        return [
            'sku' => [$required, 'string', Rule::unique('products', 'sku')->ignore($id)],
            'slug' => ['nullable', 'string', Rule::unique('products', 'slug')->ignore($id)],
            'name' => [$required, 'string', 'max:255'],
            'description' => ['nullable', 'string'],
            'brand' => ['nullable', 'string', 'max:255'],
            'unit' => ['nullable', 'string', 'max:50'],
            'category_id' => ['nullable', 'exists:categories,id'],
            'company_id' => ['nullable', 'exists:companies,id'],
            'base_price' => [$required, 'numeric', 'min:0'],
            'stock' => ['nullable', 'integer', 'min:0'],
            'min_order_qty' => ['nullable', 'integer', 'min:1'],
            'is_b2b_only' => ['boolean'],
            'is_active' => ['boolean'],
            'images' => ['nullable', 'array'],
            'images.*' => ['string', 'max:2048'],
        ];
    }
}
