-- create pizza_prices table
CREATE TABLE pizza_prices (
    id UUID,
    pizza_id UUID NOT NULL
        REFERENCES pizzas (id),
    size_id UUID NOT NULL
        REFERENCES pizza_sizes (id),
    price NUMERIC(6,2) NOT NULL
        CONSTRAINT ck_pizza_prices_price
        CHECK (price >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,

    CONSTRAINT pk_pizza_prices
        PRIMARY KEY (id),
    CONSTRAINT uq_pizza_prices_pizza_size
        UNIQUE (pizza_id, size_id)
);
