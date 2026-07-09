<?php

namespace App\Http\Requests\Admin;

use App\Enums\AnalogType;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class ProductAnalogRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        // On update the analog comes from the route, so the body only carries the pivot.
        $productId = $this->route('product')?->id;

        return [
            'analog_id' => [
                $this->isMethod('post') ? 'required' : 'prohibited',
                'integer',
                Rule::exists('products', 'id'),
                Rule::notIn([$productId]),
            ],
            'type' => ['required', Rule::in(AnalogType::values())],
            'note' => ['nullable', 'string', 'max:500'],
        ];
    }

    public function messages(): array
    {
        return [
            'analog_id.not_in' => 'A product cannot be an analog of itself.',
        ];
    }
}
