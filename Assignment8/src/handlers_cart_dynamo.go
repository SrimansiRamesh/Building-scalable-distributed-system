package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

// DynamoCart is the top-level item stored in DynamoDB.
// Items are stored as a map keyed by string product_id for atomic per-product updates.
type DynamoCart struct {
	CartID     string                `dynamodbav:"cart_id"`
	CustomerID int                   `dynamodbav:"customer_id"`
	Items      map[string]DynamoItem `dynamodbav:"items"`
}

type DynamoItem struct {
	ProductID int `dynamodbav:"product_id"`
	Quantity  int `dynamodbav:"quantity"`
}

// Response types for DynamoDB — shopping_cart_id is a UUID string instead of an int.
type CreateCartDynamoResponse struct {
	ShoppingCartID string `json:"shopping_cart_id"`
}

type GetCartDynamoResponse struct {
	ShoppingCartID string             `json:"shopping_cart_id"`
	CustomerID     int                `json:"customer_id"`
	Items          []CartItemResponse `json:"items"`
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// POST /shopping-carts
func createShoppingCartDynamo(c *gin.Context) {
	var req CreateCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
		return
	}
	if req.CustomerID < 1 {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "customer_id must be a positive integer")
		return
	}

	cartID := newUUID()
	cart := DynamoCart{
		CartID:     cartID,
		CustomerID: req.CustomerID,
		Items:      map[string]DynamoItem{},
	}

	item, err := attributevalue.MarshalMap(cart)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to marshal cart", err.Error())
		return
	}

	_, err = dynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(dynamoTable),
		Item:      item,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to create cart", err.Error())
		return
	}

	c.IndentedJSON(http.StatusCreated, CreateCartDynamoResponse{ShoppingCartID: cartID})
}

// GET /shopping-carts/:id
func getShoppingCartDynamo(c *gin.Context) {
	cartID := c.Param("id")
	if cartID == "" {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid cart ID", "cart ID cannot be empty")
		return
	}

	result, err := dynamoClient.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(dynamoTable),
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: cartID},
		},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to retrieve cart", err.Error())
		return
	}
	if result.Item == nil {
		respondError(c, http.StatusNotFound, "CART_NOT_FOUND", "Shopping cart not found",
			fmt.Sprintf("Cart with ID %s does not exist", cartID))
		return
	}

	var cart DynamoCart
	if err := attributevalue.UnmarshalMap(result.Item, &cart); err != nil {
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to parse cart", err.Error())
		return
	}

	// Enrich items with product details from in-memory seed products
	items := make([]CartItemResponse, 0, len(cart.Items))
	for _, di := range cart.Items {
		var product *Product
		for i := range SeedProducts {
			if SeedProducts[i].ProductID == di.ProductID {
				product = &SeedProducts[i]
				break
			}
		}
		if product == nil {
			continue
		}
		items = append(items, CartItemResponse{
			ProductID:    di.ProductID,
			Quantity:     di.Quantity,
			SKU:          product.SKU,
			Manufacturer: product.Manufacturer,
			CategoryID:   product.CategoryID,
			Weight:       product.Weight,
		})
	}

	c.IndentedJSON(http.StatusOK, GetCartDynamoResponse{
		ShoppingCartID: cart.CartID,
		CustomerID:     cart.CustomerID,
		Items:          items,
	})
}

// POST /shopping-carts/:id/items
func addItemsToCartDynamo(c *gin.Context) {
	cartID := c.Param("id")
	if cartID == "" {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid cart ID", "cart ID cannot be empty")
		return
	}

	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
		return
	}
	if req.ProductID < 1 || req.Quantity < 1 {
		respondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data",
			"product_id and quantity must be positive integers")
		return
	}

	// Validate product exists in seed products
	productExists := false
	for _, p := range SeedProducts {
		if p.ProductID == req.ProductID {
			productExists = true
			break
		}
	}
	if !productExists {
		respondError(c, http.StatusNotFound, "PRODUCT_NOT_FOUND", "Product not found",
			fmt.Sprintf("Product with ID %d does not exist", req.ProductID))
		return
	}

	pidKey := fmt.Sprintf("%d", req.ProductID)
	itemVal, err := attributevalue.MarshalMap(DynamoItem{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to marshal item", err.Error())
		return
	}

	_, err = dynamoClient.UpdateItem(context.Background(), &dynamodb.UpdateItemInput{
		TableName: aws.String(dynamoTable),
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: cartID},
		},
		UpdateExpression:    aws.String("SET #items.#pid = :item"),
		ConditionExpression: aws.String("attribute_exists(cart_id)"),
		ExpressionAttributeNames: map[string]string{
			"#items": "items",
			"#pid":   pidKey,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":item": &types.AttributeValueMemberM{Value: itemVal},
		},
	})
	if err != nil {
		var condFailed *types.ConditionalCheckFailedException
		if errors.As(err, &condFailed) {
			respondError(c, http.StatusNotFound, "CART_NOT_FOUND", "Shopping cart not found",
				fmt.Sprintf("Cart with ID %s does not exist", cartID))
			return
		}
		respondError(c, http.StatusInternalServerError, "DB_ERROR", "Failed to add item", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}
