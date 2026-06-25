<?php

return [
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
