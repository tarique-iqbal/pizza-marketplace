CREATE TABLE order_items (
    id       UUID,
    order_id UUID NOT NULL
        REFERENCES orders (id)
        ON DELETE CASCADE,
    pizza_id      UUID NOT NULL,
    size_id       UUID NOT NULL,
    pizza_name    VARCHAR(128) NOT NULL,
    size_diameter SMALLINT NOT NULL,
    toppings      JSONB NOT NULL DEFAULT '[]',
    quantity SMALLINT NOT NULL
        CONSTRAINT ck_order_items_quantity
        CHECK (quantity > 0),
    unit_price NUMERIC(6,2) NOT NULL
        CONSTRAINT ck_order_items_unit_price
        CHECK (unit_price >= 0),
    total_price NUMERIC(7,2) NOT NULL
        CONSTRAINT ck_order_items_total_price
        CHECK (total_price >= 0),

    CONSTRAINT pk_order_items
        PRIMARY KEY (id)
);

-- index
CREATE INDEX idx_order_items_order_id
ON order_items (order_id);
