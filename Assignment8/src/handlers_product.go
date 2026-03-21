package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func getProduct(c *gin.Context) {
    productID, err := strconv.Atoi(c.Param("productId"))
    if err != nil || productID < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_PRODUCT_ID", "Invalid product ID", "Product ID must be a positive integer")
        return
    }

    if appDB != nil {
        var p Product
        err := appDB.QueryRow(
            `SELECT product_id, sku, manufacturer, category_id, weight FROM products WHERE product_id = ?`,
            productID,
        ).Scan(&p.ProductID, &p.SKU, &p.Manufacturer, &p.CategoryID, &p.Weight)
        if err == sql.ErrNoRows {
            respondError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found", fmt.Sprintf("Product with ID %d does not exist", productID))
            return
        }
        if err != nil {
            respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
            return
        }
        c.IndentedJSON(http.StatusOK, p)
        return
    }

    value, exists := productMap.products.Load(productID)
    if !exists {
        respondError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found", fmt.Sprintf("Product with ID %d does not exist", productID))
        return
    }
    c.IndentedJSON(http.StatusOK, value.(*Product))
}

func addProductDetails(c *gin.Context) {
    productID, err := strconv.Atoi(c.Param("productId"))
    if err != nil || productID < 1 {
        respondError(c, http.StatusBadRequest, "INVALID_PRODUCT_ID", "Invalid product ID", "Product ID must be a positive integer")
        return
    }

    var product Product
    if err := c.ShouldBindJSON(&product); err != nil {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON input", err.Error())
        return
    }

    if product.SKU == "" || product.Manufacturer == "" || product.CategoryID < 1 || product.Weight < 0 {
        respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Missing or invalid required fields", "All product fields must be provided and valid")
        return
    }

    product.ProductID = productID

    if appDB != nil {
        _, err := appDB.Exec(`
            INSERT INTO products (product_id, sku, manufacturer, category_id, weight)
            VALUES (?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE
              sku = VALUES(sku),
              manufacturer = VALUES(manufacturer),
              category_id = VALUES(category_id),
              weight = VALUES(weight)`,
            product.ProductID, product.SKU, product.Manufacturer, product.CategoryID, product.Weight,
        )
        if err != nil {
            respondError(c, http.StatusInternalServerError, "DB_ERROR", "Database error", err.Error())
            return
        }
        c.Status(http.StatusNoContent)
        return
    }

    productMap.products.Store(productID, &product)
    c.Status(http.StatusNoContent)
}
