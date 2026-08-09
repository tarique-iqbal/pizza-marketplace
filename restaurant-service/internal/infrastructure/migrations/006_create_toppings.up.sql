-- global shared table
CREATE TABLE toppings (
    id UUID PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    description VARCHAR(2000),
    is_vegetarian BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_toppings_name
        UNIQUE (name)
);

INSERT INTO toppings (
    id,
    name,
    is_vegetarian
)
VALUES
('019fdcd6-ceba-76ca-9db7-e300bd6c0a27', 'Pepperoni', false),
('019fdcd6-ceba-7717-9cfd-d884c7117049', 'Mushrooms', true),
('019fdcd6-ceba-7723-80f8-fce6720fc9cb', 'Onions', true),
('019fdcd6-ceba-772d-8cb8-790c9bb15eb3', 'Extra Cheese', true),
('019fdcd6-ceba-7736-83a3-cbe944f3f92f', 'Olives', true),
('019fdcd6-ceba-7740-8c5e-b0eada511b1f', 'Bell Peppers', true),
('019fdcd6-ceba-7749-ac0f-47161e3af63a', 'Bacon', false),
('019fdcd6-ceba-7752-ad5c-510080206c08', 'Jalapeños', true),
('019fdce7-fced-71d2-b243-13d9adce8ffb', 'Barbecue Sauce', true),
('019fdce7-fced-7228-9f8b-574da86b2f3e', 'Minced Meat', false),
('019fdce7-fced-7234-a06b-6f5601ad33cc', 'Corn', true),
('019fdce7-fced-723d-874a-f158ebf4c790', 'Tuna', false)
ON CONFLICT (name) DO NOTHING;
