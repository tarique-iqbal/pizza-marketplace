CREATE TABLE carts (
    id            UUID,
    customer_id   UUID NOT NULL,
    restaurant_id UUID NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ,

    CONSTRAINT pk_carts
        PRIMARY KEY (id),
    CONSTRAINT uq_carts_customer_id
        UNIQUE (customer_id)
);
