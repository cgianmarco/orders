INSERT INTO vat_categories (rate, name) VALUES
    (22, 'standard rate'),
    (10, 'reduced rate'),
    (5, 'special reduced rate'),
    (4, 'super reduced rate')
ON CONFLICT (rate) DO NOTHING;

INSERT INTO items (name, quantityInStock, priceCents, vatCategoryId) VALUES
    ('Laptop', 10, 99999, 1),
    ('Mouse', 10, 2550, 1),
    ('Keyboard', 10, 7500, 1),
    ('Monitor', 10, 29999, 1),
    ('Webcam', 10, 8999, 1),
    ('Headphones', 10, 14999, 1),
    ('USB Cable', 10, 1299, 1),
    ('External SSD', 10, 17999, 1),
    ('Desk Lamp', 10, 4550, 1),
    ('Phone Stand', 10, 1999, 1);