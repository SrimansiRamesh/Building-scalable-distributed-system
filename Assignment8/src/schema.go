package main

// SeedProducts are preloaded on app startup (DB or in-memory).
var SeedProducts = []Product{
	{ProductID: 1, SKU: "DEMO-SKU-001", Manufacturer: "Demo Manufacturing", CategoryID: 1, Weight: 42},
	{ProductID: 2, SKU: "DEMO-SKU-002", Manufacturer: "Demo Manufacturing", CategoryID: 1, Weight: 55},
	{ProductID: 3, SKU: "WIDGET-100", Manufacturer: "Acme Corp", CategoryID: 2, Weight: 120},
	{ProductID: 4, SKU: "GADGET-200", Manufacturer: "Tech Industries", CategoryID: 2, Weight: 85},
	{ProductID: 5, SKU: "TOOL-301", Manufacturer: "BuildRight", CategoryID: 3, Weight: 250},
}

const createProductsTable = `
CREATE TABLE IF NOT EXISTS products (
  product_id INT PRIMARY KEY,
  sku VARCHAR(512) NOT NULL,
  manufacturer VARCHAR(512) NOT NULL,
  category_id INT NOT NULL,
  weight INT NOT NULL
)`

const createShoppingCartsTable = `
CREATE TABLE IF NOT EXISTS shopping_carts (
  id INT AUTO_INCREMENT PRIMARY KEY,
  customer_id INT NOT NULL
)`

const createShoppingCartItemsTable = `
CREATE TABLE IF NOT EXISTS shopping_cart_items (
  cart_id INT NOT NULL,
  product_id INT NOT NULL,
  quantity INT NOT NULL,
  PRIMARY KEY (cart_id, product_id),
  CONSTRAINT chk_cart_items_quantity CHECK (quantity >= 1)
)`

