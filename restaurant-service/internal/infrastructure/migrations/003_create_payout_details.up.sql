-- create enum: payout_details_status_enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'payout_details_status_enum'
    ) THEN
        CREATE TYPE payout_details_status_enum AS ENUM (
            'pending',
            'active',
            'superseded'
        );
    END IF;
END$$;

-- create payout_details table
CREATE TABLE payout_details (
    id UUID,
    restaurant_id UUID NOT NULL
        REFERENCES restaurants (id)
        ON DELETE CASCADE,
    account_holder VARCHAR(100) NOT NULL,
    iban VARCHAR(34) NOT NULL,
    bic VARCHAR(11) NOT NULL,
    bank_name VARCHAR(100) NOT NULL,
    status payout_details_status_enum NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,

    CONSTRAINT pk_payout_details
        PRIMARY KEY (id)
);

-- indexes
CREATE INDEX idx_payout_details_restaurant_id
ON payout_details (restaurant_id);

-- at most one active payout record per restaurant
CREATE UNIQUE INDEX uq_payout_details_restaurant_active
ON payout_details (restaurant_id)
WHERE status = 'active';

-- at most one pending (unapproved) payout record per restaurant
CREATE UNIQUE INDEX uq_payout_details_restaurant_pending
ON payout_details (restaurant_id)
WHERE status = 'pending';
