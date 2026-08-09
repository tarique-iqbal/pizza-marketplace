CREATE TABLE topping_prices (
    id UUID,
    restaurant_id UUID NOT NULL
        REFERENCES restaurants (id)
        ON DELETE CASCADE,
    topping_id UUID NOT NULL
        REFERENCES toppings (id),
    extra_price NUMERIC(6,2) NOT NULL
        CONSTRAINT ck_topping_prices_extra_price
        CHECK (extra_price >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,

    CONSTRAINT pk_topping_prices
        PRIMARY KEY (id),
    CONSTRAINT uq_topping_prices_restaurant_topping
        UNIQUE (restaurant_id, topping_id)
);

CREATE INDEX idx_topping_prices_restaurant_id
    ON topping_prices (restaurant_id);
