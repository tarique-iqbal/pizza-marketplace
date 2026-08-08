-- global shared table
CREATE TABLE pizza_sizes (
    id UUID PRIMARY KEY,
    diameter_cm SMALLINT NOT NULL
        CONSTRAINT ck_pizza_sizes_diameter_cm
        CHECK (diameter_cm BETWEEN 20 AND 45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
        CONSTRAINT uq_pizza_sizes_diameter_cm
        UNIQUE (diameter_cm)
);

INSERT INTO pizza_sizes (
    id,
    diameter_cm
)
VALUES
('019e3c11-b1db-75f7-8057-419bf0ed7181', 24),
('019e3c11-b1db-7335-b63e-93b529beac8a', 28),
('019e3c11-b1db-7c58-bdd7-42caf426d47d', 32),
('019e3c11-b1db-7713-b56d-67eb1eeadcf3', 45)
ON CONFLICT (diameter_cm) DO NOTHING;
