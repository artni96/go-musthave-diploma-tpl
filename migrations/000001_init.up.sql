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
    number VARCHAR(255) NOT NULL UNIQUE,
    uploaded_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE balance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    current INTEGER,
    withdrawn INTEGER,
    CONSTRAINT fk_user FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE 
)