package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func createShoppingCart(c *gin.Context) {
    if appDB == nil {
        respondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Shopping cart requires database", "")
        return
    }
    var req CreateCartRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON input", err.Error())
        return
    }
    if req.CustomerID < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "customer_id must be a positive integer")
        return
    }
    result, err := appDB.Exec(`INSERT INTO shopping_carts (customer_id) VALUES (?)`, req.CustomerID)
    if err != nil {
        log.Printf("createShoppingCart: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    id, err := result.LastInsertId()
    if err != nil {
        log.Printf("createShoppingCart LastInsertId: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    c.IndentedJSON(http.StatusCreated, CreateCartResponse{ShoppingCartID: int(id)})
}

func getShoppingCart(c *gin.Context) {
    if appDB == nil {
        respondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Shopping cart requires database", "")
        return
    }
    cartID, err := strconv.Atoi(c.Param("id"))
    if err != nil || cartID < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid cart ID", "Shopping cart ID must be a positive integer")
        return
    }
    // Single query: cart + items + product details via JOINs
    rows, err := appDB.Query(`
        SELECT c.id, c.customer_id,
               i.product_id, i.quantity,
               p.sku, p.manufacturer, p.category_id, p.weight
        FROM shopping_carts c
        LEFT JOIN shopping_cart_items i ON c.id = i.cart_id
        LEFT JOIN products p ON i.product_id = p.product_id
        WHERE c.id = ?`, cartID)
    if err != nil {
        log.Printf("getShoppingCart: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    defer rows.Close()

    var cartIDOut, customerID int
    var items []CartItemResponse
    var haveCart bool
    rowIndex := 0
    for rows.Next() {
        var productID, quantity, categoryID, weight sql.NullInt64
        var sku, manufacturer sql.NullString
        if err := rows.Scan(&cartIDOut, &customerID, &productID, &quantity, &sku, &manufacturer, &categoryID, &weight); err != nil {
            log.Printf("getShoppingCart scan: %v", err)
            respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
            return
        }
        rowIndex++
        haveCart = true
        // LEFT JOIN yields one row per item; product columns are NULL when cart has no items
        if productID.Valid && quantity.Valid && sku.Valid && manufacturer.Valid && categoryID.Valid && weight.Valid {
            items = append(items, CartItemResponse{
                ProductID:    int(productID.Int64),
                Quantity:     int(quantity.Int64),
                SKU:          sku.String,
                Manufacturer: manufacturer.String,
                CategoryID:   int(categoryID.Int64),
                Weight:       int(weight.Int64),
            })
        }
    }
    if err := rows.Err(); err != nil {
        log.Printf("getShoppingCart rows: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    if !haveCart {
        respondError(c, http.StatusNotFound, "CART_NOT_FOUND", "Shopping cart not found", fmt.Sprintf("Cart with ID %d does not exist", cartID))
        return
    }
    if items == nil {
        items = []CartItemResponse{}
    }
    c.IndentedJSON(http.StatusOK, GetCartResponse{ShoppingCartID: cartIDOut, CustomerID: customerID, Items: items})
}

func addItemsToCart(c *gin.Context) {
    if appDB == nil {
        respondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Shopping cart requires database", "")
        return
    }
    cartID, err := strconv.Atoi(c.Param("id"))
    if err != nil || cartID < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid cart ID", "Shopping cart ID must be a positive integer")
        return
    }
    var req AddCartItemRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON input", err.Error())
        return
    }
    if req.ProductID < 1 || req.Quantity < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "product_id and quantity must be positive integers")
        return
    }
    tx, err := appDB.Begin()
    if err != nil {
        log.Printf("addItemsToCart Begin: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    defer func() { _ = tx.Rollback() }()

    // Check if the cart exists
    var exists int
    err = tx.QueryRow(`SELECT 1 FROM shopping_carts WHERE id = ?`, cartID).Scan(&exists)
    if err == sql.ErrNoRows {
        respondError(c, http.StatusNotFound, "CART_NOT_FOUND", "Shopping cart not found", fmt.Sprintf("Cart with ID %d does not exist", cartID))
        return
    }
    if err != nil {
        log.Printf("addItemsToCart cart check: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }

    // Check if the product exists
    err = tx.QueryRow(`SELECT 1 FROM products WHERE product_id = ?`, req.ProductID).Scan(&exists)
    if err == sql.ErrNoRows {
        respondError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found", fmt.Sprintf("Product with ID %d does not exist", req.ProductID))
        return
    }
    if err != nil {
        log.Printf("addItemsToCart product check: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }

    // Add the item to the cart if it doesn't exist or update the quantity if it does
    _, err = tx.Exec(`
        INSERT INTO shopping_cart_items (cart_id, product_id, quantity) VALUES (?, ?, ?)
        ON DUPLICATE KEY UPDATE quantity = VALUES(quantity)`, cartID, req.ProductID, req.Quantity)
    if err != nil {
        log.Printf("addItemsToCart insert: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }

    // Commit the transaction
    if err := tx.Commit(); err != nil {
        log.Printf("addItemsToCart Commit: %v", err)
        respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", "")
        return
    }
    c.Status(http.StatusNoContent)
}
