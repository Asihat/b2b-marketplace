-- Marketplace schema. Mirrors the former Laravel migrations one to one so the
-- Go service can run against a database those migrations created; every
-- statement is idempotent for that reason.

CREATE TABLE IF NOT EXISTS companies (
    id bigserial PRIMARY KEY,
    name varchar(255) NOT NULL,
    slug varchar(255) NOT NULL UNIQUE,
    tax_number varchar(255),
    email varchar(255),
    phone varchar(255),
    country varchar(2),
    address text,
    default_currency varchar(3) NOT NULL DEFAULT 'USD',
    default_locale varchar(5) NOT NULL DEFAULT 'en',
    is_verified boolean NOT NULL DEFAULT false,
    created_at timestamp(0),
    updated_at timestamp(0)
);

CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    name varchar(255) NOT NULL,
    email varchar(255) NOT NULL UNIQUE,
    email_verified_at timestamp(0),
    password varchar(255) NOT NULL,
    remember_token varchar(100),
    created_at timestamp(0),
    updated_at timestamp(0),
    company_id bigint REFERENCES companies(id) ON DELETE SET NULL,
    type varchar(255) NOT NULL DEFAULT 'b2c',
    role varchar(255) NOT NULL DEFAULT 'customer',
    phone varchar(255),
    locale varchar(5) NOT NULL DEFAULT 'en',
    currency varchar(3) NOT NULL DEFAULT 'USD',
    is_active boolean NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS users_company_id_index ON users (company_id);

CREATE TABLE IF NOT EXISTS personal_access_tokens (
    id bigserial PRIMARY KEY,
    tokenable_type varchar(255) NOT NULL,
    tokenable_id bigint NOT NULL,
    name text NOT NULL,
    token varchar(64) NOT NULL UNIQUE,
    abilities text,
    last_used_at timestamp(0),
    expires_at timestamp(0),
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS personal_access_tokens_tokenable_type_tokenable_id_index
    ON personal_access_tokens (tokenable_type, tokenable_id);
CREATE INDEX IF NOT EXISTS personal_access_tokens_expires_at_index ON personal_access_tokens (expires_at);

CREATE TABLE IF NOT EXISTS currencies (
    id bigserial PRIMARY KEY,
    code varchar(3) NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    symbol varchar(8) NOT NULL,
    exchange_rate numeric(18, 8) NOT NULL DEFAULT 1,
    is_base boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp(0),
    updated_at timestamp(0)
);

CREATE TABLE IF NOT EXISTS languages (
    id bigserial PRIMARY KEY,
    code varchar(5) NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    native_name varchar(255) NOT NULL,
    is_default boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp(0),
    updated_at timestamp(0)
);

CREATE TABLE IF NOT EXISTS categories (
    id bigserial PRIMARY KEY,
    parent_id bigint REFERENCES categories(id) ON DELETE SET NULL,
    slug varchar(255) NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    name_translations json,
    position integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS categories_parent_id_index ON categories (parent_id);

CREATE TABLE IF NOT EXISTS products (
    id bigserial PRIMARY KEY,
    category_id bigint REFERENCES categories(id) ON DELETE SET NULL,
    company_id bigint REFERENCES companies(id) ON DELETE SET NULL,
    sku varchar(255) NOT NULL UNIQUE,
    slug varchar(255) NOT NULL UNIQUE,
    name varchar(255) NOT NULL,
    description text,
    brand varchar(255),
    unit varchar(255) NOT NULL DEFAULT 'pcs',
    base_price numeric(18, 4) NOT NULL DEFAULT 0,
    stock integer NOT NULL DEFAULT 0,
    min_order_qty integer NOT NULL DEFAULT 1,
    is_b2b_only boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS products_brand_index ON products (brand);
CREATE INDEX IF NOT EXISTS products_category_id_index ON products (category_id);
CREATE INDEX IF NOT EXISTS products_company_id_index ON products (company_id);

CREATE TABLE IF NOT EXISTS product_translations (
    id bigserial PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    locale varchar(5) NOT NULL,
    name varchar(255) NOT NULL,
    description text,
    created_at timestamp(0),
    updated_at timestamp(0),
    UNIQUE (product_id, locale)
);

CREATE TABLE IF NOT EXISTS product_prices (
    id bigserial PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    currency_code varchar(3) NOT NULL,
    min_qty integer NOT NULL DEFAULT 1,
    price numeric(18, 4) NOT NULL,
    created_at timestamp(0),
    updated_at timestamp(0),
    UNIQUE (product_id, currency_code, min_qty)
);

CREATE TABLE IF NOT EXISTS product_analogs (
    id bigserial PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    analog_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    type varchar(255) NOT NULL DEFAULT 'equivalent',
    note text,
    created_at timestamp(0),
    updated_at timestamp(0),
    UNIQUE (product_id, analog_id)
);
CREATE INDEX IF NOT EXISTS product_analogs_analog_id_index ON product_analogs (analog_id);

CREATE TABLE IF NOT EXISTS product_images (
    id bigserial PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url varchar(255) NOT NULL,
    alt varchar(255),
    position integer NOT NULL DEFAULT 0,
    is_primary boolean NOT NULL DEFAULT false,
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS product_images_product_id_position_index ON product_images (product_id, position);

CREATE TABLE IF NOT EXISTS orders (
    id bigserial PRIMARY KEY,
    number varchar(255) NOT NULL UNIQUE,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id bigint REFERENCES companies(id) ON DELETE SET NULL,
    type varchar(255) NOT NULL DEFAULT 'b2c',
    status varchar(255) NOT NULL DEFAULT 'pending',
    currency_code varchar(3) NOT NULL,
    subtotal numeric(18, 4) NOT NULL DEFAULT 0,
    tax_total numeric(18, 4) NOT NULL DEFAULT 0,
    grand_total numeric(18, 4) NOT NULL DEFAULT 0,
    shipping_address text,
    notes text,
    created_at timestamp(0),
    updated_at timestamp(0)
);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS contact_name varchar(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS contact_email varchar(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS contact_phone varchar(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_city varchar(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_postal_code varchar(255);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS shipping_country varchar(2);
CREATE INDEX IF NOT EXISTS orders_user_id_index ON orders (user_id);
CREATE INDEX IF NOT EXISTS orders_company_id_index ON orders (company_id);
CREATE INDEX IF NOT EXISTS orders_created_at_index ON orders (created_at);
CREATE INDEX IF NOT EXISTS orders_status_index ON orders (status);

CREATE TABLE IF NOT EXISTS order_items (
    id bigserial PRIMARY KEY,
    order_id bigint NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id bigint REFERENCES products(id) ON DELETE SET NULL,
    name varchar(255) NOT NULL,
    sku varchar(255) NOT NULL,
    quantity integer NOT NULL,
    unit_price numeric(18, 4) NOT NULL,
    line_total numeric(18, 4) NOT NULL,
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS order_items_order_id_index ON order_items (order_id);
CREATE INDEX IF NOT EXISTS order_items_product_id_index ON order_items (product_id);

CREATE TABLE IF NOT EXISTS payments (
    id bigserial PRIMARY KEY,
    order_id bigint NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    gateway varchar(255) NOT NULL,
    status varchar(255) NOT NULL DEFAULT 'pending',
    currency_code varchar(3) NOT NULL,
    amount numeric(18, 4) NOT NULL,
    reference varchar(255),
    payload json,
    paid_at timestamp(0),
    created_at timestamp(0),
    updated_at timestamp(0)
);
CREATE INDEX IF NOT EXISTS payments_order_id_index ON payments (order_id);
CREATE INDEX IF NOT EXISTS payments_reference_index ON payments (reference);

CREATE TABLE IF NOT EXISTS app_settings (
    id bigserial PRIMARY KEY,
    key varchar(255) NOT NULL UNIQUE,
    value text,
    created_at timestamp(0),
    updated_at timestamp(0)
);
