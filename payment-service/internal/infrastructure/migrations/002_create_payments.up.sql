-- create enum: payment_status_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'payment_status_enum'
    ) THEN
        CREATE TYPE payment_status_enum AS ENUM (
            'pending',
            'succeeded',
            'failed'
        );
    END IF;
END$$;

-- create payments table
CREATE TABLE payments (
    id                 UUID          NOT NULL,
    subject_type       VARCHAR(32)   NOT NULL,
    subject_id         UUID          NOT NULL,
    restaurant_id      UUID          NOT NULL,
    customer_id        UUID          NOT NULL,
    amount             NUMERIC(10,2) NOT NULL CHECK (amount >= 0),
    currency           VARCHAR(3)    NOT NULL,
    platform_fee       NUMERIC(10,2) NOT NULL DEFAULT 0,
    status             payment_status_enum NOT NULL DEFAULT 'pending',
    gateway            VARCHAR(32)   NOT NULL,
    gateway_payment_id VARCHAR(64),
    failure_reason     VARCHAR(500),
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMPTZ,

    CONSTRAINT pk_payments
        PRIMARY KEY (id),

    CONSTRAINT uq_payments_subject
        UNIQUE (subject_type, subject_id)
);

-- webhook lookup index
CREATE INDEX idx_payments_gateway_payment_id
ON payments (gateway_payment_id);
