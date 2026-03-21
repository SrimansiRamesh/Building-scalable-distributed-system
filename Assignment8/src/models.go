package main

import "sync"

// Data models and request/response types

type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type CreateCartRequest struct {
	CustomerID int `json:"customer_id"`
}

type CreateCartResponse struct {
	ShoppingCartID int `json:"shopping_cart_id"`
}

type CartItemResponse struct {
	ProductID    int    `json:"product_id"`
	Quantity     int    `json:"quantity"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
}

type GetCartResponse struct {
	ShoppingCartID int               `json:"shopping_cart_id"`
	CustomerID     int               `json:"customer_id"`
	Items          []CartItemResponse `json:"items"`
}

type AddCartItemRequest struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// In-memory fallback when DB_HOST is unset
type ProductMap struct {
	products sync.Map
}

var productMap = &ProductMap{}

