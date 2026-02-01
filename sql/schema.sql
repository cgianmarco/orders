CREATE TABLE IF NOT EXISTS vat_categories (
    id SERIAL PRIMARY KEY,
    rate INT NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    quantityInStock INT NOT NULL DEFAULT 0,
    priceCents BIGINT NOT NULL,
    vatCategoryId INT NOT NULL,
    FOREIGN KEY (vatCategoryId) REFERENCES vat_categories(id)
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    createdAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    orderId INT NOT NULL,
    itemId INT NOT NULL,
    quantity INT NOT NULL,
    priceCents BIGINT NOT NULL,
    vatCents BIGINT NOT NULL,
    PRIMARY KEY (orderId, itemId),
    FOREIGN KEY (orderId) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (itemId) REFERENCES items(id)
);