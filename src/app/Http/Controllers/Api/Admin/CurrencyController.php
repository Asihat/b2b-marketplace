<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Currency;
use App\Services\CurrencyService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class CurrencyController extends Controller
{
    public function __construct(protected CurrencyService $currencies) {}

    public function index(): JsonResponse
    {
        return response()->json(Currency::orderByDesc('is_base')->orderBy('code')->get());
    }

    public function store(Request $request): JsonResponse
    {
        $data = $request->validate([
            'code' => ['required', 'string', 'size:3', 'unique:currencies,code'],
            'name' => ['required', 'string', 'max:255'],
            'symbol' => ['required', 'string', 'max:8'],
            'exchange_rate' => ['required', 'numeric', 'min:0'],
            'is_base' => ['boolean'],
            'is_active' => ['boolean'],
        ]);

        $currency = Currency::create($this->normalizeBase($data));
        $this->currencies->flush();

        return response()->json($currency, 201);
    }

    public function update(Request $request, Currency $currency): JsonResponse
    {
        $data = $request->validate([
            'code' => ['sometimes', 'string', 'size:3', Rule::unique('currencies', 'code')->ignore($currency->id)],
            'name' => ['sometimes', 'string', 'max:255'],
            'symbol' => ['sometimes', 'string', 'max:8'],
            'exchange_rate' => ['sometimes', 'numeric', 'min:0'],
            'is_base' => ['boolean'],
            'is_active' => ['boolean'],
        ]);

        $currency->update($this->normalizeBase($data));
        $this->currencies->flush();

        return response()->json($currency->fresh());
    }

    public function destroy(Currency $currency): JsonResponse
    {
        abort_if($currency->is_base, 422, 'Cannot delete the base currency.');

        $currency->delete();
        $this->currencies->flush();

        return response()->json(['message' => 'Currency deleted.']);
    }

    /** Ensure only one base currency exists, pinned to rate 1.0. */
    protected function normalizeBase(array $data): array
    {
        if (! empty($data['is_base'])) {
            Currency::where('is_base', true)->update(['is_base' => false]);
            $data['exchange_rate'] = 1.0;
        }
        if (isset($data['code'])) {
            $data['code'] = strtoupper($data['code']);
        }

        return $data;
    }
}
