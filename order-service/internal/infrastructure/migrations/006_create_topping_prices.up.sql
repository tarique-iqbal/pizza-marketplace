CREATE TABLE topping_prices (
    restaurant_id UUID NOT NULL
        REFERENCES restaurants (id)
        ON DELETE CASCADE,
    topping_id  UUID NOT NULL,
    name        VARCHAR(255) NOT NULL,
    extra_price NUMERIC(6,2) NOT NULL
        CONSTRAINT ck_topping_prices_extra_price
        CHECK (extra_price >= 0),
    updated_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT pk_topping_prices
        PRIMARY KEY (restaurant_id, topping_id)
);
