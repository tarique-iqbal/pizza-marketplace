CREATE TABLE pizza_prices (
    pizza_id    UUID NOT NULL
        REFERENCES pizzas (id)
        ON DELETE CASCADE,
    size_id     UUID NOT NULL,
    diameter_cm SMALLINT NOT NULL,
    price NUMERIC(6,2) NOT NULL
        CONSTRAINT ck_pizza_prices_price
        CHECK (price >= 0),
    is_active   BOOLEAN NOT NULL DEFAULT true,
    updated_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_pizza_prices
        PRIMARY KEY (pizza_id, size_id)
);
