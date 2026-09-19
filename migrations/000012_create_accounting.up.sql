CREATE TABLE accounting_books (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL CHECK (name <> ''),
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_books_user_id
    ON accounting_books (user_id, id);

CREATE TABLE accounting_expenses (
    id           BIGSERIAL PRIMARY KEY,
    book_id      BIGINT      NOT NULL REFERENCES accounting_books(id) ON DELETE CASCADE,
    amount_cents BIGINT      NOT NULL CHECK (amount_cents > 0),
    category     SMALLINT    NOT NULL CHECK (category IN (1, 2, 3, 4, 5, 6, 7, 8)),
    spent_at     TIMESTAMPTZ NOT NULL,
    note         VARCHAR(200) NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_accounting_expenses_book_spent_id
    ON accounting_expenses (book_id, spent_at DESC, id DESC);

COMMENT ON COLUMN accounting_expenses.category IS
    '支出分类：1餐饮，2交通，3住宿，4购物，5票务，6娱乐，7医疗，8其他';
