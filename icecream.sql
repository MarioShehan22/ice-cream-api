CREATE DATABASE IF NOT EXISTS icecream_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE icecream_test;

-- ---------------------------------------------------------------
-- Schema
-- ---------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
                                     id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                     name          VARCHAR(100)    NOT NULL,
                                     email         VARCHAR(150)    NOT NULL,
                                     password_hash VARCHAR(100)    NOT NULL,
                                     role          VARCHAR(20)     NOT NULL,
                                     created_at    DATETIME(3)     NULL,
                                     updated_at    DATETIME(3)     NULL,
                                     PRIMARY KEY (id),
                                     UNIQUE KEY idx_users_email (email),
                                     KEY idx_users_role (role),
                                     CONSTRAINT chk_users_role CHECK (role IN ('owner', 'manager', 'cashier'))
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS categories (
                                          id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                          name       VARCHAR(100)    NOT NULL,
                                          created_at DATETIME(3)     NULL,
                                          PRIMARY KEY (id),
                                          UNIQUE KEY idx_categories_name (name)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS items (
                                     id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                     name        VARCHAR(150)    NOT NULL,
                                     description VARCHAR(1000)   NULL,
                                     category_id BIGINT UNSIGNED NOT NULL,
                                     price       DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
                                     quantity    INT             NOT NULL DEFAULT 0,
                                     created_at  DATETIME(3)     NULL,
                                     updated_at  DATETIME(3)     NULL,
                                     PRIMARY KEY (id),
                                     KEY idx_items_name (name),
                                     KEY idx_items_category_id (category_id),
                                     CONSTRAINT fk_items_category FOREIGN KEY (category_id)
                                         REFERENCES categories (id) ON UPDATE CASCADE ON DELETE RESTRICT,
                                     CONSTRAINT chk_items_price CHECK (price >= 0),
                                     CONSTRAINT chk_items_quantity CHECK (quantity >= 0)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS customers (
                                         id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
                                         name       VARCHAR(100)    NOT NULL,
                                         email      VARCHAR(150)    NOT NULL,
                                         phone      VARCHAR(20)     NOT NULL,
                                         address    VARCHAR(255)    NULL,
                                         created_at DATETIME(3)     NULL,
                                         updated_at DATETIME(3)     NULL,
                                         PRIMARY KEY (id),
                                         UNIQUE KEY idx_customers_email (email),
                                         KEY idx_customers_name (name)
) ENGINE=InnoDB;

-- JWT IDs of logged-out tokens (kept until they expire)
CREATE TABLE IF NOT EXISTS revoked_tokens (
                                              id         VARCHAR(64) NOT NULL,
                                              expires_at DATETIME(3) NOT NULL,
                                              PRIMARY KEY (id),
                                              KEY idx_revoked_tokens_expires_at (expires_at)
) ENGINE=InnoDB;

-- ---------------------------------------------------------------
-- Seed data
-- ---------------------------------------------------------------

INSERT IGNORE INTO users
(name, email, password_hash, role, created_at, updated_at)
VALUES
    ('Kasun Perera',   'owner@icecream.test',   '$2b$10$dVcg8CHTKs.xtoNZG91Hne5vcp4WDAJoEyv60YCaUKGUWjUItKLdO', 'owner',   NOW(3), NOW(3)),
    ('Nuwan Silva',    'manager@icecream.test', '$2b$10$xs5hMWkqpGDXo8ZvTHWsVOXPE/e0lNg3DeND4vhvwjJd7tse6SBx2', 'manager', NOW(3), NOW(3)),
    ('Amali Fernando', 'cashier@icecream.test', '$2b$10$mBfV9Jfbnq0.hK8IB/ApJ.757F9HwhCNU968CKsJqaoBkl6jlHZIS', 'cashier', NOW(3), NOW(3)),
    ('Shehan Silva',   'shehan@gmail.com',      '$2b$10$B7y5adZNMlYBWiQvaItNmO9HvB9IAXBIfFRe9cjo0eeDsUMOO3cnO', 'owner',   NOW(3), NOW(3));

INSERT IGNORE INTO categories (name, created_at) VALUES
                                                     ('Chocolate',  NOW(3)),
                                                     ('Vanilla',    NOW(3)),
                                                     ('Strawberry', NOW(3));

INSERT INTO items
(name, description, category_id, price, quantity, created_at, updated_at)
SELECT
    v.name,
    v.description,
    c.id,
    v.price,
    v.quantity,
    NOW(3),
    NOW(3)
FROM (
         SELECT 'Double Chocolate Scoop' AS name,
                'Rich dark chocolate' AS description,
                'Chocolate' AS category, 2.50 AS price, 40 AS quantity
         UNION ALL
         SELECT 'Choco Fudge Sundae',
                'Chocolate with fudge sauce',
                'Chocolate', 4.75, 25
         UNION ALL
         SELECT 'Classic Vanilla Scoop',
                'Madagascar vanilla',
                'Vanilla', 2.25, 60
         UNION ALL
         SELECT 'Vanilla Bean Cone',
                'Vanilla in a waffle cone',
                'Vanilla', 3.25, 35
         UNION ALL
         SELECT 'Strawberry Swirl Scoop',
                'Fresh strawberry swirl',
                'Strawberry', 2.50, 30
         UNION ALL
         SELECT 'Strawberry Milkshake',
                'Thick strawberry shake',
                'Strawberry', 5.00, 20
     ) AS v
         JOIN categories c ON c.name = v.category
WHERE NOT EXISTS (SELECT 1 FROM items);

INSERT IGNORE INTO customers
(name, email, phone, address, created_at, updated_at)
VALUES
    ('Tharushi Perera', 'tharushi@example.com',
     '0771234567', '12 Main Street, Negombo', NOW(3), NOW(3)),
    ('Kasuni Jayawardena', 'kasuni@example.com',
     '0712345678', '45 Lake Road, Kandy', NOW(3), NOW(3)),
    ('Dinesh Fernando', 'dinesh@example.com',
     '0753456789', '7 Temple Road, Colombo', NOW(3), NOW(3));

