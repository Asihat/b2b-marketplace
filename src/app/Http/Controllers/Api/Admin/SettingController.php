<?php

namespace App\Http\Controllers\Api\Admin;

use App\Http\Controllers\Controller;
use App\Services\MarketplaceSettings;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class SettingController extends Controller
{
    public function __construct(private readonly MarketplaceSettings $settings) {}

    public function index(): JsonResponse
    {
        return response()->json($this->settings->publicSettings());
    }

    public function update(Request $request): JsonResponse
    {
        $data = $request->validate([
            'mode' => ['sometimes', 'required', 'string', Rule::in(['b2c', 'b2b'])],
            'company_name' => ['sometimes', 'required', 'string', 'max:120'],
            'company_description' => ['sometimes', 'nullable', 'string', 'max:500'],
        ]);

        return response()->json($this->settings->update($data));
    }

    public function uploadIcon(Request $request): JsonResponse
    {
        $data = $request->validate([
            'icon' => ['required', 'image', 'mimes:png,jpg,jpeg,webp', 'max:2048'],
        ]);

        $path = $data['icon']->store('settings/icons', 'public');
        abort_if($path === false, 500, 'Icon could not be stored.');

        return response()->json($this->settings->updateIcon($path));
    }

    public function removeIcon(): JsonResponse
    {
        return response()->json($this->settings->removeIcon());
    }
}
