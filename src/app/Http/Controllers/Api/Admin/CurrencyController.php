<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\CurrencyRequest;
use App\Models\Currency;
use App\Services\CurrencyService;
use Illuminate\Http\JsonResponse;

class CurrencyController extends Controller
{
    public function __construct(private readonly CurrencyService $currencies) {}

    public function index(): JsonResponse
    {
        return response()->json(Currency::orderByDesc('is_base')->orderBy('code')->get());
    }

    public function store(CurrencyRequest $request): JsonResponse
    {
        $currency = Currency::create($this->normalize($request->validated()));
        $this->currencies->flush();

        return response()->json($currency, 201);
    }

    public function update(CurrencyRequest $request, Currency $currency): JsonResponse
    {
        $currency->update($this->normalize($request->validated()));
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

    /** Ensure only one base currency exists, pinned to rate 1.0, code upper-cased. */
    private function normalize(array $data): array
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
