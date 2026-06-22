<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class StoreOrderRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'items' => ['required', 'array', 'min:1'],
            'items.*.product_id' => ['required', 'integer', 'exists:products,id'],
            'items.*.quantity' => ['required', 'integer', 'min:1'],
            'currency_code' => ['nullable', 'string', 'size:3'],
            'contact_name' => ['nullable', 'string', 'max:255'],
            'contact_email' => ['nullable', 'email', 'max:255'],
            'contact_phone' => ['nullable', 'string', 'max:50'],
            'shipping_address' => ['required', 'string', 'max:1000'],
            'shipping_city' => ['nullable', 'string', 'max:255'],
            'shipping_postal_code' => ['nullable', 'string', 'max:32'],
            'shipping_country' => ['nullable', 'string', 'size:2'],
            'notes' => ['nullable', 'string', 'max:2000'],
            'tax_rate' => ['nullable', 'numeric', 'min:0', 'max:1'],
        ];
    }
}
