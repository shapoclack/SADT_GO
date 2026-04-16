CREATE TABLE products (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    price REAL NOT NULL,
    count INTEGER NOT NULL
);
INSERT INTO products (title, price, count) VALUES ('Keyboard MadLion', 120.5, 10);
INSERT INTO products (title, price, count) VALUES ('Mouse', 45.0, 30);


DROP TABLE products;