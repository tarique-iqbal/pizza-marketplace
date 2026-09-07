-- create enum: order_status_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'order_status_enum'
    ) THEN
        CREATE TYPE order_status_enum AS ENUM (
            'pending',
            'confirmed',
            'ready',
            'completed',
            'cancelled'
        );
    END IF;
END$$;

-- create enum: order_fulfillment_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'order_fulfillment_enum'
    ) THEN
        CREATE TYPE order_fulfillment_enum AS ENUM (
            'delivery',
            'pickup'
        );
    END IF;
END$$;

-- create orders table
CREATE TABLE orders (
    id               UUID,
    customer_id      UUID NOT NULL,
    restaurant_id    UUID NOT NULL,
    status           order_status_enum NOT NULL DEFAULT 'pending',
    fulfillment      order_fulfillment_enum NOT NULL,
    contact_email    VARCHAR(255) NOT NULL,
    contact_phone    VARCHAR(32),
    delivery_address JSONB,
    delivery_lat DOUBLE PRECISION
        CONSTRAINT ck_orders_delivery_lat
        CHECK (delivery_lat BETWEEN -90 AND 90),
    delivery_lon DOUBLE PRECISION
        CONSTRAINT ck_orders_delivery_lon
        CHECK (delivery_lon BETWEEN -180 AND 180),
    subtotal NUMERIC(8,2) NOT NULL
        CONSTRAINT ck_orders_subtotal
        CHECK (subtotal >= 0),
    delivery_fee NUMERIC(5,2) NOT NULL DEFAULT 0
        CONSTRAINT ck_orders_delivery_fee
        CHECK (delivery_fee >= 0),
    total NUMERIC(8,2) NOT NULL
        CONSTRAINT ck_orders_total
        CHECK (total >= 0),
    currency     CHAR(3) NOT NULL DEFAULT 'EUR',
    payment_id   VARCHAR(64),
    placed_at    TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    ready_at     TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,

    CONSTRAINT pk_orders
        PRIMARY KEY (id),
    CONSTRAINT ck_orders_total_matches_subtotal_plus_fee
        CHECK (total = subtotal + delivery_fee)
);

-- indexes
CREATE INDEX idx_orders_customer_id
ON orders (customer_id);

CREATE INDEX idx_orders_restaurant_id
ON orders (restaurant_id);

-- at most one order per payment_id, only when set
CREATE UNIQUE INDEX uq_orders_payment_id
ON orders (payment_id)
WHERE payment_id IS NOT NULL;
