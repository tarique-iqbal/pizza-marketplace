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

-- create restaurants table
CREATE TABLE restaurants (
    id            UUID,
    owner_id      UUID NOT NULL,
    name          VARCHAR(128) NOT NULL,
    owner_email   VARCHAR(255) NOT NULL,
    lat DOUBLE PRECISION NOT NULL
        CONSTRAINT ck_restaurants_lat
        CHECK (lat BETWEEN -90 AND 90),
    lon DOUBLE PRECISION NOT NULL
        CONSTRAINT ck_restaurants_lon
        CHECK (lon BETWEEN -180 AND 180),
    delivery_km SMALLINT
        CONSTRAINT ck_restaurants_delivery_km
        CHECK (delivery_km BETWEEN 1 AND 25),
    delivery_fee NUMERIC(5,2) NOT NULL DEFAULT 0.00
        CONSTRAINT ck_restaurants_delivery_fee
        CHECK (delivery_fee >= 0),
    minimum_order NUMERIC(6,2) NOT NULL DEFAULT 0.00
        CONSTRAINT ck_restaurants_minimum_order
        CHECK (minimum_order >= 0),
    pickup BOOLEAN NOT NULL DEFAULT true,
    delivery_type restaurant_delivery_type_enum
        NOT NULL DEFAULT 'none',
    currency   CHAR(3) NOT NULL DEFAULT 'EUR',
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_restaurants
        PRIMARY KEY (id)
);
