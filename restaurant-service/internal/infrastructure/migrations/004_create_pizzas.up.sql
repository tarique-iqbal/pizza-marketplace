-- create enum: pizza_status_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'pizza_status_enum'
    ) THEN
        CREATE TYPE pizza_status_enum AS ENUM (
            'available',
            'unavailable',
            'archived'
        );
    END IF;
END$$;

-- create pizzas table
CREATE TABLE pizzas (
    id UUID,
    restaurant_id UUID NOT NULL
        REFERENCES restaurants (id)
        ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(255),
    is_vegetarian BOOLEAN NOT NULL DEFAULT false,
    status pizza_status_enum NOT NULL DEFAULT 'available',
    sort_order INTEGER NOT NULL DEFAULT 0
        CONSTRAINT ck_pizzas_sort_order
        CHECK (sort_order >= 0),
    toppings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,

    CONSTRAINT pk_pizzas
        PRIMARY KEY (id)
);

-- indexes
CREATE INDEX idx_pizzas_restaurant_id
ON pizzas (restaurant_id);
