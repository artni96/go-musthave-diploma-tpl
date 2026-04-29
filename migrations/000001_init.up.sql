CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(60) NOT NULL
);

CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    accrual INTEGER,
    status VARCHAR(255) NOT NULL ,
    order_number VARCHAR(255) NOT NULL UNIQUE,
    uploaded_at TIMESTAMPTZ NOT NULL
);
