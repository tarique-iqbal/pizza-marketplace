-- create enum: restaurant_delivery_type_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'restaurant_delivery_type_enum'
    ) THEN
        CREATE TYPE restaurant_delivery_type_enum AS ENUM (
            'own',
            'external',
            'none'
        );
    END IF;
END$$;

-- create enum: restaurant_status_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'restaurant_status_enum'
    ) THEN
        CREATE TYPE restaurant_status_enum AS ENUM (
            'draft',
            'review',
            'approved',
            'active',
            'inactive',
            'disabled',
            'rejected'
        );
    END IF;
END$$;

-- create restaurants table
CREATE TABLE restaurants (
    id UUID,
    owner_id UUID NOT NULL,
    name VARCHAR(128) NOT NULL,
    vat_number VARCHAR(16) NOT NULL,
    slug VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(32),
    website VARCHAR(255),
    checklist JSONB NOT NULL DEFAULT '{}'::jsonb,
    status restaurant_status_enum NOT NULL DEFAULT 'draft',
    address JSONB NOT NULL DEFAULT '{}'::jsonb,
    lat DOUBLE PRECISION
        CONSTRAINT ck_restaurants_lat
        CHECK (lat BETWEEN -90 AND 90),
    lon DOUBLE PRECISION
        CONSTRAINT ck_restaurants_lon
        CHECK (lon BETWEEN -180 AND 180),
    opening_hours JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    pickup BOOLEAN NOT NULL DEFAULT true,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    delivery_km SMALLINT
        CONSTRAINT ck_restaurants_delivery_km
        CHECK (delivery_km BETWEEN 1 AND 25),
    delivery_type restaurant_delivery_type_enum
        NOT NULL DEFAULT 'none',
    delivery_fee NUMERIC(5,2) NOT NULL DEFAULT 0.00
        CONSTRAINT ck_restaurants_delivery_fee
        CHECK (delivery_fee >= 0),
    minimum_order NUMERIC(6,2) NOT NULL DEFAULT 0.00
        CONSTRAINT ck_restaurants_minimum_order
        CHECK (minimum_order >= 0),
    rating NUMERIC(2,1) NOT NULL DEFAULT 0
        CONSTRAINT ck_restaurants_rating
        CHECK (rating BETWEEN 0 AND 5),
    total_reviews INTEGER NOT NULL DEFAULT 0
        CONSTRAINT ck_restaurants_total_reviews
        CHECK (total_reviews >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,

    CONSTRAINT pk_restaurants
        PRIMARY KEY (id)
);

-- unique indexes
CREATE UNIQUE INDEX uq_restaurants_slug
ON restaurants (slug)
WHERE slug IS NOT NULL;

CREATE UNIQUE INDEX uq_restaurants_email
ON restaurants (email)
WHERE email IS NOT NULL;

CREATE UNIQUE INDEX uq_restaurants_phone
ON restaurants (phone)
WHERE phone IS NOT NULL;
