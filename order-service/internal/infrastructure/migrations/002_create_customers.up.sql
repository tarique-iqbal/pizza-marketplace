CREATE TABLE customers (
    id         UUID,
    email      VARCHAR(255) NOT NULL,
    first_name VARCHAR(128) NOT NULL,
    phone      VARCHAR(32),

    CONSTRAINT pk_customers
        PRIMARY KEY (id)
);
