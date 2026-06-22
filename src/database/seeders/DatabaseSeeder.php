<?php

namespace Database\Seeders;

use App\Models\Category;
use App\Models\Company;
use App\Models\Currency;
use App\Models\Language;
use App\Models\Product;
use App\Models\User;
use Illuminate\Database\Seeder;
use Illuminate\Support\Str;

class DatabaseSeeder extends Seeder
{
    public function run(): void
    {
        $this->seedCurrencies();
        $this->seedLanguages();
        $categories = $this->seedCategories();
        $company = $this->seedCompanyAndUsers();
        $this->seedAnchorProducts($categories, $company);
        $this->seedCatalog($categories, $company);
    }

    protected function seedCurrencies(): void
    {
        $currencies = [
            ['code' => 'USD', 'name' => 'US Dollar', 'symbol' => '$', 'exchange_rate' => 1.0, 'is_base' => true],
            ['code' => 'EUR', 'name' => 'Euro', 'symbol' => '€', 'exchange_rate' => 0.92, 'is_base' => false],
            ['code' => 'KZT', 'name' => 'Kazakhstani Tenge', 'symbol' => '₸', 'exchange_rate' => 470.0, 'is_base' => false],
            ['code' => 'RUB', 'name' => 'Russian Ruble', 'symbol' => '₽', 'exchange_rate' => 90.0, 'is_base' => false],
        ];

        foreach ($currencies as $c) {
            Currency::updateOrCreate(['code' => $c['code']], $c);
        }
    }

    protected function seedLanguages(): void
    {
        $languages = [
            ['code' => 'en', 'name' => 'English', 'native_name' => 'English', 'is_default' => true],
            ['code' => 'ru', 'name' => 'Russian', 'native_name' => 'Русский', 'is_default' => false],
            ['code' => 'kk', 'name' => 'Kazakh', 'native_name' => 'Қазақша', 'is_default' => false],
        ];

        foreach ($languages as $l) {
            Language::updateOrCreate(['code' => $l['code']], $l);
        }
    }

    /** @return array<string, Category> */
    protected function seedCategories(): array
    {
        $defs = [
            'electronics' => ['en' => 'Electronics', 'ru' => 'Электроника', 'kk' => 'Электроника'],
            'industrial' => ['en' => 'Industrial Supplies', 'ru' => 'Промышленные товары', 'kk' => 'Өнеркәсіптік тауарлар'],
            'office' => ['en' => 'Office', 'ru' => 'Офис', 'kk' => 'Кеңсе'],
            'tools' => ['en' => 'Tools', 'ru' => 'Инструменты', 'kk' => 'Құралдар'],
            'packaging' => ['en' => 'Packaging', 'ru' => 'Упаковка', 'kk' => 'Қаптама'],
            'safety' => ['en' => 'Safety & PPE', 'ru' => 'Средства защиты', 'kk' => 'Қорғаныс құралдары'],
        ];

        $out = [];
        $pos = 0;
        foreach ($defs as $slug => $names) {
            $out[$slug] = Category::updateOrCreate(
                ['slug' => $slug],
                ['name' => $names['en'], 'name_translations' => $names, 'position' => $pos++],
            );
        }

        return $out;
    }

    protected function seedCompanyAndUsers(): Company
    {
        $company = Company::updateOrCreate(
            ['slug' => 'acme-trading'],
            [
                'name' => 'Acme Trading LLC',
                'tax_number' => 'TX-1000-2000',
                'country' => 'KZ',
                'default_currency' => 'USD',
                'default_locale' => 'en',
                'is_verified' => true,
            ]
        );

        User::updateOrCreate(
            ['email' => 'admin@marketplace.test'],
            ['name' => 'Marketplace Admin', 'password' => 'password', 'type' => 'b2b', 'role' => 'admin', 'company_id' => $company->id],
        );
        User::updateOrCreate(
            ['email' => 'buyer@acme.test'],
            ['name' => 'Acme Buyer', 'password' => 'password', 'type' => 'b2b', 'role' => 'customer', 'company_id' => $company->id, 'currency' => 'USD'],
        );
        User::updateOrCreate(
            ['email' => 'customer@example.test'],
            ['name' => 'Retail Customer', 'password' => 'password', 'type' => 'b2c', 'role' => 'customer', 'currency' => 'EUR'],
        );

        return $company;
    }

    /**
     * Hand-crafted products that carry multi-language translations and a couple
     * of explicit analog links — used by the i18n/analog walkthroughs.
     *
     * @param array<string, Category> $categories
     */
    protected function seedAnchorProducts(array $categories, Company $company): void
    {
        $anchors = [
            [
                'sku' => 'CBL-USB-C-1M', 'name' => 'USB-C Cable 1m', 'brand' => 'Volt', 'category' => 'electronics',
                'base_price' => 4.50, 'stock' => 5000, 'tr' => ['ru' => 'Кабель USB-C 1м', 'kk' => 'USB-C кабелі 1м'],
            ],
            [
                'sku' => 'CBL-USB-C-1M-PRO', 'name' => 'USB-C Cable 1m (Braided)', 'brand' => 'Volt', 'category' => 'electronics',
                'base_price' => 6.90, 'stock' => 3000, 'tr' => ['ru' => 'Кабель USB-C 1м (оплётка)', 'kk' => 'USB-C кабелі 1м (өрімді)'],
            ],
            [
                'sku' => 'IND-BEARING-608', 'name' => 'Ball Bearing 608ZZ', 'brand' => 'RollTech', 'category' => 'industrial',
                'base_price' => 0.80, 'stock' => 100000, 'min_order_qty' => 100, 'is_b2b_only' => true,
                'tr' => ['ru' => 'Подшипник 608ZZ', 'kk' => 'Мойынтірек 608ZZ'],
            ],
            [
                'sku' => 'IND-BEARING-608-2RS', 'name' => 'Ball Bearing 608-2RS', 'brand' => 'RollTech', 'category' => 'industrial',
                'base_price' => 0.95, 'stock' => 80000, 'min_order_qty' => 100, 'is_b2b_only' => true,
                'tr' => ['ru' => 'Подшипник 608-2RS', 'kk' => 'Мойынтірек 608-2RS'],
            ],
        ];

        $made = [];
        foreach ($anchors as $a) {
            $product = $this->makeProduct($company, [
                'category_id' => $categories[$a['category']]->id,
                'sku' => $a['sku'],
                'name' => $a['name'],
                'brand' => $a['brand'],
                'base_price' => $a['base_price'],
                'stock' => $a['stock'],
                'min_order_qty' => $a['min_order_qty'] ?? 1,
                'is_b2b_only' => $a['is_b2b_only'] ?? false,
            ]);

            foreach ($a['tr'] as $locale => $name) {
                $product->translations()->updateOrCreate(['locale' => $locale], ['name' => $name, 'description' => $name]);
            }
            $made[$a['sku']] = $product;
        }

        $this->linkAnalogs($made['CBL-USB-C-1M'], $made['CBL-USB-C-1M-PRO'], 'upgrade', 'Braided variant');
        $this->linkAnalogs($made['IND-BEARING-608'], $made['IND-BEARING-608-2RS'], 'equivalent', 'Sealed version');
    }

    /**
     * Generates a broad catalog. Each "family" produces several variants that
     * are automatically linked to each other as analogs.
     *
     * @param array<string, Category> $categories
     */
    protected function seedCatalog(array $categories, Company $company): void
    {
        // [category, brand, unit, b2bOnly, minOrderQty, base, [ [variant, price, stock], ... ]]
        $families = [
            ['electronics', 'Volt', 'pcs', false, 1, 'HDMI 2.1 Cable', [['1m', 7.5, 4000], ['2m', 9.9, 3500], ['3m', 12.5, 2000]]],
            ['electronics', 'Volt', 'pcs', false, 1, 'USB-C Power Adapter', [['30W', 14, 1500], ['65W', 19, 1200], ['100W', 29, 800]]],
            ['electronics', 'Klyk', 'pcs', false, 1, 'Wireless Mouse', [['Basic', 8.9, 2200], ['Ergo', 15.5, 1400], ['Pro', 24.9, 900]]],
            ['electronics', 'Klyk', 'pcs', false, 1, 'Mechanical Keyboard', [['TKL Red', 39, 600], ['Full Brown', 45, 500], ['Wireless Blue', 59, 350]]],
            ['electronics', 'Datacore', 'pcs', false, 1, 'NVMe SSD', [['512GB', 42, 700], ['1TB', 69, 500], ['2TB', 119, 250]]],
            ['electronics', 'Voltbank', 'pcs', false, 1, 'Power Bank', [['10000mAh', 18, 1300], ['20000mAh', 27, 900]]],
            ['electronics', 'Klyk', 'pcs', false, 1, 'USB Hub', [['4-Port', 11, 1800], ['7-Port', 17, 1100]]],
            ['electronics', 'SeeCam', 'pcs', false, 1, 'Webcam', [['720p', 16, 800], ['1080p', 26, 650], ['4K', 49, 300]]],

            ['industrial', 'RollTech', 'pcs', true, 100, 'Hex Bolt M8', [['20mm', 0.12, 200000], ['30mm', 0.15, 180000], ['40mm', 0.18, 150000]]],
            ['industrial', 'RollTech', 'pcs', true, 100, 'Ball Bearing 6203', [['ZZ', 1.1, 60000], ['2RS', 1.3, 55000]]],
            ['industrial', 'FlowLine', 'm', true, 10, 'Hydraulic Hose 1/2"', [['SAE 1SN', 4.2, 9000], ['SAE 2SN', 5.6, 7000]]],
            ['industrial', 'BeltPro', 'pcs', true, 5, 'V-Belt A-Section', [['A40', 3.1, 12000], ['A50', 3.6, 10000], ['A60', 4.0, 9000]]],
            ['industrial', 'SealMaster', 'set', true, 5, 'Gasket Set', [['Small', 6.5, 4000], ['Medium', 9.0, 3000], ['Large', 12.0, 2000]]],
            ['industrial', 'ArcWeld', 'kg', true, 5, 'Welding Rod E6013', [['2.5mm', 2.2, 20000], ['3.2mm', 2.5, 18000]]],
            ['industrial', 'PipeCo', 'm', true, 6, 'Steel Pipe DN50', [['1.5mm wall', 7.8, 8000], ['2.0mm wall', 9.4, 6000]]],

            ['office', 'PaperCo', 'ream', false, 1, 'A4 Paper 80g', [['White', 5.2, 20000], ['Recycled', 5.8, 9000], ['Premium', 6.9, 7000]]],
            ['office', 'InkFlow', 'pcs', false, 1, 'Ballpoint Pen', [['Blue', 0.4, 50000], ['Black', 0.4, 50000], ['Red', 0.4, 30000]]],
            ['office', 'Clipix', 'pcs', false, 1, 'Stapler', [['Mini', 3.2, 4000], ['Desktop', 5.5, 3000], ['Heavy Duty', 11.0, 1200]]],
            ['office', 'StikIt', 'pack', false, 1, 'Sticky Notes', [['Yellow 3x3', 1.2, 15000], ['Neon 3x3', 1.6, 9000]]],
            ['office', 'InkFlow', 'pcs', false, 1, 'Whiteboard Marker', [['Black', 0.9, 12000], ['Assorted 4-pack', 3.2, 6000]]],
            ['office', 'Filewise', 'pcs', false, 1, 'Lever Arch File', [['A4 Blue', 1.8, 9000], ['A4 Black', 1.8, 9000]]],

            ['tools', 'DriveMaster', 'pcs', false, 1, 'Cordless Drill', [['12V', 38, 700], ['18V', 59, 500], ['18V Brushless', 89, 250]]],
            ['tools', 'GripPro', 'set', false, 1, 'Screwdriver Set', [['6-piece', 7.5, 2500], ['12-piece', 12.9, 1500]]],
            ['tools', 'GripPro', 'pcs', false, 1, 'Claw Hammer', [['16oz', 6.9, 3000], ['20oz', 8.5, 2200]]],
            ['tools', 'MeasureX', 'pcs', false, 1, 'Measuring Tape', [['5m', 3.5, 5000], ['8m', 5.0, 3500]]],
            ['tools', 'GripPro', 'set', false, 1, 'Combination Wrench Set', [['8-19mm', 18, 1200], ['8-22mm', 24, 900]]],
            ['tools', 'CutEdge', 'pcs', false, 1, 'Angle Grinder', [['115mm', 32, 800], ['125mm', 39, 600]]],

            ['packaging', 'BoxIt', 'pcs', false, 1, 'Cardboard Box', [['S 20x15x10', 0.45, 40000], ['M 30x20x15', 0.7, 30000], ['L 40x30x20', 1.1, 20000]]],
            ['packaging', 'WrapCo', 'roll', false, 1, 'Bubble Wrap Roll', [['50cm x 50m', 6.5, 5000], ['100cm x 50m', 11.0, 3000]]],
            ['packaging', 'TapeWorks', 'pcs', false, 1, 'Packing Tape', [['Clear 48mm', 0.9, 30000], ['Brown 48mm', 0.9, 25000]]],
            ['packaging', 'WrapCo', 'roll', false, 1, 'Stretch Film', [['Standard', 4.8, 8000], ['Heavy', 6.2, 6000]]],
            ['packaging', 'BoxIt', 'pack', false, 1, 'Padded Mailer', [['A5', 2.4, 12000], ['A4', 3.6, 9000]]],

            ['safety', 'SafeGuard', 'pcs', false, 1, 'Safety Goggles', [['Clear', 2.2, 9000], ['Anti-Fog', 3.4, 6000]]],
            ['safety', 'HandPro', 'pair', false, 1, 'Work Gloves', [['Nitrile', 1.5, 20000], ['Cut-Resistant', 3.8, 8000], ['Leather', 5.2, 5000]]],
            ['safety', 'SafeGuard', 'pcs', false, 1, 'Hard Hat', [['White', 6.5, 4000], ['Yellow', 6.5, 4000]]],
            ['safety', 'HiVis', 'pcs', false, 1, 'Hi-Vis Vest', [['Yellow M', 3.2, 7000], ['Orange L', 3.2, 7000]]],
            ['safety', 'BreatheWell', 'pcs', false, 1, 'Respirator Mask', [['FFP2', 0.8, 50000], ['FFP3', 1.2, 30000]]],
        ];

        foreach ($families as [$cat, $brand, $unit, $b2bOnly, $moq, $base, $variants]) {
            $famProducts = [];
            $famKey = strtoupper(preg_replace('/[^A-Z0-9]/', '', strtoupper(Str::ascii($base))));
            $famKey = substr($famKey, 0, 10);

            foreach ($variants as $i => [$variant, $price, $stock]) {
                $product = $this->makeProduct($company, [
                    'category_id' => $categories[$cat]->id,
                    'sku' => $famKey.'-'.($i + 1),
                    'name' => "{$base} {$variant}",
                    'brand' => $brand,
                    'unit' => $unit,
                    'base_price' => $price,
                    'stock' => $stock,
                    'min_order_qty' => $moq,
                    'is_b2b_only' => $b2bOnly,
                ]);
                $famProducts[] = $product;
            }

            // Link every variant in the family to the others as analogs.
            foreach ($famProducts as $a) {
                foreach ($famProducts as $b) {
                    if ($a->id !== $b->id) {
                        $a->analogs()->syncWithoutDetaching([$b->id => ['type' => 'equivalent', 'note' => "Variant of {$base}"]]);
                    }
                }
            }
        }
    }

    /** Create/update a product, attach a volume tier and images. */
    protected function makeProduct(Company $company, array $attrs): Product
    {
        $product = Product::updateOrCreate(
            ['sku' => $attrs['sku']],
            [
                'category_id' => $attrs['category_id'],
                'company_id' => $company->id,
                'slug' => Str::slug($attrs['name'].' '.$attrs['sku']),
                'name' => $attrs['name'],
                'description' => $attrs['name'].' — supplied by '.$company->name.'. Quality '.strtolower($attrs['name']).' for B2B and retail buyers.',
                'brand' => $attrs['brand'],
                'unit' => $attrs['unit'] ?? 'pcs',
                'base_price' => $attrs['base_price'],
                'stock' => $attrs['stock'],
                'min_order_qty' => $attrs['min_order_qty'] ?? 1,
                'is_b2b_only' => $attrs['is_b2b_only'] ?? false,
                'is_active' => true,
            ]
        );

        // B2B volume tier: 15% off per unit at 500+.
        if (($attrs['is_b2b_only'] ?? false) || ($attrs['min_order_qty'] ?? 1) > 1) {
            $product->prices()->updateOrCreate(
                ['currency_code' => 'USD', 'min_qty' => 500],
                ['price' => round($attrs['base_price'] * 0.85, 4)],
            );
        }

        $this->attachImages($product);

        return $product;
    }

    /**
     * Attach deterministic placeholder images served by the app itself
     * (see ImageController) — no external image host required.
     */
    protected function attachImages(Product $product, int $count = 3): void
    {
        if ($product->images()->exists()) {
            return;
        }

        for ($i = 1; $i <= $count; $i++) {
            $product->images()->create([
                'url' => url("/img/{$product->slug}-{$i}"),
                'alt' => $product->name,
                'position' => $i,
                'is_primary' => $i === 1,
            ]);
        }
    }

    protected function linkAnalogs(Product $a, Product $b, string $type, ?string $note = null): void
    {
        $a->analogs()->syncWithoutDetaching([$b->id => ['type' => $type, 'note' => $note]]);
        $b->analogs()->syncWithoutDetaching([$a->id => ['type' => $type, 'note' => $note]]);
    }
}
