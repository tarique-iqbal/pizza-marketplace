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
    id            UUID,
    restaurant_id UUID NOT NULL
        REFERENCES restaurants (id)
        ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    status        pizza_status_enum NOT NULL DEFAULT 'available',
    updated_at    TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_pizzas
        PRIMARY KEY (id)
);

-- index
CREATE INDEX idx_pizzas_restaurant_id
ON pizzas (restaurant_id);
