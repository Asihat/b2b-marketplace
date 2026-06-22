<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\CompanyRequest;
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

    public function update(CompanyRequest $request, Company $company): JsonResponse
    {
        $company->update($request->validated());

        return response()->json($company->fresh());
    }
}
