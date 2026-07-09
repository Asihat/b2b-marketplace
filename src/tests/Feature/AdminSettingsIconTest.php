<?php

namespace Tests\Feature;

use App\Enums\UserRole;
use App\Models\AppSetting;
use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Http\UploadedFile;
use Illuminate\Support\Facades\Storage;
use Laravel\Sanctum\Sanctum;
use Tests\TestCase;

class AdminSettingsIconTest extends TestCase
{
    use RefreshDatabase;

    public function test_admin_can_upload_the_main_icon(): void
    {
        Storage::fake('public');
        Sanctum::actingAs(User::factory()->create(['role' => UserRole::Admin]));

        $response = $this->postJson('/api/admin/settings/icon', [
            'icon' => UploadedFile::fake()->createWithContent(
                'icon.png',
                base64_decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII='),
            ),
        ]);

        $path = AppSetting::where('key', 'main_icon_path')->value('value');

        $response
            ->assertOk()
            ->assertJsonPath('icon_url', Storage::disk('public')->url($path));
        Storage::disk('public')->assertExists($path);
    }

    public function test_icon_upload_rejects_non_images(): void
    {
        Storage::fake('public');
        Sanctum::actingAs(User::factory()->create(['role' => UserRole::Admin]));

        $this->postJson('/api/admin/settings/icon', [
            'icon' => UploadedFile::fake()->create('icon.txt', 10, 'text/plain'),
        ])->assertUnprocessable()->assertJsonValidationErrors('icon');
    }

    public function test_admin_can_remove_the_main_icon_and_its_file(): void
    {
        Storage::fake('public');
        Storage::disk('public')->put('settings/icons/icon.png', 'image');
        AppSetting::updateOrCreate(
            ['key' => 'main_icon_path'],
            ['value' => 'settings/icons/icon.png'],
        );
        Sanctum::actingAs(User::factory()->create(['role' => UserRole::Admin]));

        $this->deleteJson('/api/admin/settings/icon')
            ->assertOk()
            ->assertJsonPath('icon_url', null);

        $this->assertDatabaseMissing('app_settings', ['key' => 'main_icon_path']);
        Storage::disk('public')->assertMissing('settings/icons/icon.png');
    }

    public function test_admin_can_update_company_details(): void
    {
        Sanctum::actingAs(User::factory()->create(['role' => UserRole::Admin]));

        $this->putJson('/api/admin/settings', [
            'company_name' => 'Acme Marketplace',
            'company_description' => 'Wholesale supplies for growing businesses.',
        ])->assertOk()
            ->assertJsonPath('company_name', 'Acme Marketplace')
            ->assertJsonPath('company_description', 'Wholesale supplies for growing businesses.');

        $this->assertDatabaseHas('app_settings', [
            'key' => 'company_name',
            'value' => 'Acme Marketplace',
        ]);
    }
}
