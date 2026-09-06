CREATE TABLE cart_items (
    id       UUID,
    cart_id  UUID NOT NULL
        REFERENCES carts (id)
        ON DELETE CASCADE,
    pizza_id UUID NOT NULL,
    size_id  UUID NOT NULL,
    quantity SMALLINT NOT NULL
        CONSTRAINT ck_cart_items_quantity
        CHECK (quantity > 0),
    toppings JSONB NOT NULL DEFAULT '[]',

    CONSTRAINT pk_cart_items
        PRIMARY KEY (id),
    CONSTRAINT uq_cart_items_cart_pizza_size_toppings
        UNIQUE (cart_id, pizza_id, size_id, toppings)
);
