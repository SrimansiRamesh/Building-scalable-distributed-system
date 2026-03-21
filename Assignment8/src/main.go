package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var appDB *sql.DB
var dynamoClient *dynamodb.Client
var dynamoTable string

func respondError(c *gin.Context, status int, code, message, details string) {
	c.IndentedJSON(status, ErrorResponse{
		Error:   code,
		Message: message,
		Details: details,
	})
}

func openMySQL() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return nil, nil
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	if user == "" || name == "" {
		return nil, fmt.Errorf("DB_USER and DB_NAME are required when DB_HOST is set")
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, pass, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(createProductsTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	if _, err := db.Exec(createShoppingCartsTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	if _, err := db.Exec(createShoppingCartItemsTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("schema: %w", err)
	}
	log.Println("Connected to MySQL; products and shopping_carts tables ready")
	return db, nil
}

func preloadProducts() {
	if appDB != nil {
		for _, p := range SeedProducts {
			_, err := appDB.Exec(`
				INSERT INTO products (product_id, sku, manufacturer, category_id, weight)
				VALUES (?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE
					sku = VALUES(sku),
					manufacturer = VALUES(manufacturer),
					category_id = VALUES(category_id),
					weight = VALUES(weight)`,
				p.ProductID, p.SKU, p.Manufacturer, p.CategoryID, p.Weight)
			if err != nil {
				log.Printf("preload product %d: %v", p.ProductID, err)
			}
		}
		log.Printf("Preloaded %d products into DB", len(SeedProducts))
	} else {
		for i := range SeedProducts {
			p := &SeedProducts[i]
			productMap.products.Store(p.ProductID, p)
		}
		log.Printf("Preloaded %d products into memory", len(SeedProducts))
	}
}

func initDynamo() error {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("load AWS config: %w", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
	dynamoTable = os.Getenv("DYNAMODB_TABLE_NAME")
	if dynamoTable == "" {
		dynamoTable = "shopping-carts"
	}
	log.Printf("DynamoDB client ready; table=%s region=%s", dynamoTable, region)
	return nil
}

func main() {
	backend := os.Getenv("STORAGE_BACKEND")

	if backend == "dynamodb" {
		if err := initDynamo(); err != nil {
			log.Fatalf("DynamoDB: %v", err)
		}
	} else {
		db, err := openMySQL()
		if err != nil {
			log.Fatalf("MySQL: %v", err)
		}
		appDB = db
		if appDB != nil {
			defer appDB.Close()
		} else {
			log.Println("DB_HOST unset; using in-memory product store")
		}
	}
	preloadProducts()

	r := gin.Default()
	r.GET("/products/:productId", getProduct)
	r.POST("/products/:productId/details", addProductDetails)

	if backend == "dynamodb" {
		r.POST("/shopping-carts", createShoppingCartDynamo)
		r.GET("/shopping-carts/:id", getShoppingCartDynamo)
		r.POST("/shopping-carts/:id/items", addItemsToCartDynamo)
	} else {
		r.POST("/shopping-carts", createShoppingCart)
		r.GET("/shopping-carts/:id", getShoppingCart)
		r.POST("/shopping-carts/:id/items", addItemsToCart)
	}

	r.GET("/tables/:tableName", getTableRows)
	r.GET("/health", func(c *gin.Context) {
		h := gin.H{"status": "healthy"}
		switch backend {
		case "dynamodb":
			h["db"] = "dynamodb"
			h["table"] = dynamoTable
		default:
			if appDB != nil {
				h["db"] = "mysql"
			} else {
				h["db"] = "memory"
			}
		}
		c.JSON(http.StatusOK, h)
	})

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
