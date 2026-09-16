-- create customer_addresses table
CREATE TABLE customer_addresses (
    id UUID,
    customer_id UUID NOT NULL
        REFERENCES customers (id)
        ON DELETE CASCADE,
    house VARCHAR(20) NOT NULL,
    street VARCHAR(255) NOT NULL,
    city VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,

    CONSTRAINT pk_customer_addresses
        PRIMARY KEY (id)
);

-- indexes
CREATE INDEX idx_customer_addresses_customer_id
ON customer_addresses (customer_id);

-- at most one default address per customer
CREATE UNIQUE INDEX uq_customer_addresses_default
ON customer_addresses (customer_id)
WHERE is_default;
