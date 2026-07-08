<?php

return [
    'company_name' => env('MARKETPLACE_COMPANY_NAME', 'Marketplace'),
    'company_description' => env('MARKETPLACE_COMPANY_DESCRIPTION', 'B2B and B2C marketplace'),

    /*
    |--------------------------------------------------------------------------
    | Marketplace Mode
    |--------------------------------------------------------------------------
    |
    | Supported values: b2c, b2b. In B2B mode storefront product surfaces show
    | seller company names where company data exists.
    |
    */
    'mode' => env('MARKETPLACE_MODE', 'b2b'),
];
