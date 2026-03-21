-- Test table + sample rows (schema only; app preloads products on startup)
CREATE TABLE IF NOT EXISTS products (
  product_id INT PRIMARY KEY,
  sku VARCHAR(512) NOT NULL,
  manufacturer VARCHAR(512) NOT NULL,
  category_id INT NOT NULL,
  weight INT NOT NULL
);

CREATE TABLE IF NOT EXISTS shopping_carts (
  id INT AUTO_INCREMENT PRIMARY KEY,
  customer_id INT NOT NULL
);

CREATE TABLE IF NOT EXISTS shopping_cart_items (
  cart_id INT NOT NULL,
  product_id INT NOT NULL,
  quantity INT NOT NULL,
  PRIMARY KEY (cart_id, product_id),
  FOREIGN KEY (cart_id) REFERENCES shopping_carts(id) ON DELETE CASCADE,
  FOREIGN KEY (product_id) REFERENCES products(product_id),
  CONSTRAINT chk_quantity CHECK (quantity >= 1)
);

INSERT INTO products (product_id, sku, manufacturer, category_id, weight) VALUES
  (1, 'DEMO-SKU-001', 'Demo Manufacturing', 1, 42),
  (2, 'DEMO-SKU-002', 'Demo Manufacturing', 1, 55)
ON DUPLICATE KEY UPDATE
  sku = VALUES(sku),
  manufacturer = VALUES(manufacturer),
  category_id = VALUES(category_id),
  weight = VALUES(weight);
