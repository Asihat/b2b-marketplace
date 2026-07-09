<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Services\DashboardService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class DashboardController extends Controller
{
    public function __construct(private readonly DashboardService $dashboard) {}

    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'days' => ['sometimes', 'integer', Rule::in(DashboardService::RANGES)],
        ]);

        return response()->json(
            $this->dashboard->overview((int) ($validated['days'] ?? DashboardService::DEFAULT_RANGE))
        );
    }
}
