-- Monetary amounts are stored as BIGINT minor units (paise): 1234.50 is stored as 123450.
-- This avoids floating point rounding errors in financial calculations.

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(100)  NOT NULL,
    email         VARCHAR(254)  NOT NULL,
    password_hash VARCHAR(255)  NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email_lower ON users (LOWER(email));

CREATE TABLE transactions (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type             VARCHAR(10)   NOT NULL CHECK (type IN ('income', 'expense')),
    category         VARCHAR(50)   NOT NULL,
    amount           BIGINT        NOT NULL CHECK (amount > 0),
    description      VARCHAR(255)  NOT NULL DEFAULT '',
    transaction_date DATE          NOT NULL,
    payment_method   VARCHAR(30)   NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_user_date ON transactions (user_id, transaction_date);

CREATE INDEX idx_transactions_user_type_category ON transactions (user_id, type, category);

CREATE TABLE budgets (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   VARCHAR(50)  NOT NULL,
    amount     BIGINT       NOT NULL CHECK (amount > 0),
    month      SMALLINT     NOT NULL CHECK (month BETWEEN 1 AND 12),
    year       SMALLINT     NOT NULL CHECK (year BETWEEN 2000 AND 2100),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_budget_user_category_month UNIQUE (user_id, category, month, year)
)
