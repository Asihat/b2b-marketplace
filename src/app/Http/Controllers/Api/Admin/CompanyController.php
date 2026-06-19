<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Models\Company;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class CompanyController extends Controller
{
    public function index(Request $request): JsonResponse
    {
        $companies = Company::query()
            ->withCount(['users', 'products', 'orders'])
            ->when($request->query('search'), fn ($q, $s) => $q->where('name', 'ilike', "%{$s}%"))
            ->latest()
            ->paginate(20);

        return response()->json($companies);
    }

    public function update(Request $request, Company $company): JsonResponse
    {
        $data = $request->validate([
            'name' => ['sometimes', 'string', 'max:255'],
            'tax_number' => ['nullable', 'string', 'max:100'],
            'email' => ['nullable', 'email'],
            'phone' => ['nullable', 'string', 'max:50'],
            'country' => ['nullable', 'string', 'size:2'],
            'address' => ['nullable', 'string'],
            'default_currency' => ['nullable', 'string', 'size:3'],
            'default_locale' => ['nullable', 'string', 'max:5'],
            'is_verified' => ['boolean'],
        ]);

        $company->update($data);

        return response()->json($company->fresh());
    }
}
