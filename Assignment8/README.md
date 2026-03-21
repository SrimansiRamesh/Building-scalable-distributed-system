## Installation


### Docker Setup

1. Build and run the Docker image:
```
docker build -t hw8 .
docker run -p 8080:8080 hw8
```



### Terraform Setup
####  Prepare Credentials

Retrieve you temporary credentials from Learner's Lab.
Enter your configuration when prompted:
```
aws configure
aws configure set aws_session_token <YOUR-TEMP-SESSION-TOKEBN>
```

#### Apply Infrastructure
```
cd terraform
terraform init
terraform apply -auto-approve
```

ECS tasks receive MySQL settings as environment variables: `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` (from Terraform / RDS).



#### Clean Up
```
terraform destroy -auto-approve
```




--- 





## Locust

The purpose of Locust is not to stress test, but rather an easier way of response time.

```
cd locust
docker-compose up
```

Run in background with `docker-compose up -d`

Stop all services `docker-compose down`







# Plotting performance results

This requires that you've run the performance test already (from `locust.py`) and saved the response times in a json.

```
python3 plot_response_times.py
python3 plot_response_times.py --input results/other.json --output results/custom.png --seed 7
```








# Example endpoints

### Products

```
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-001",
    "manufacturer": "Acme Corp",
    "category_id": 10,
    "weight": 500
  }'
```

```
curl http://localhost:8080/products/1
```

### Shopping cart

Create a cart (requires DB; returns `shopping_cart_id`):

```
curl -X POST http://localhost:8080/shopping-carts \
  -H "Content-Type: application/json" \
  -d '{"customer_id": 1}'
```

Get a cart and its items:

```
curl http://localhost:8080/shopping-carts/1
```

Add items to a cart (product must exist in `products`):

```
curl -X POST http://localhost:8080/shopping-carts/1/items \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1, "quantity": 2}'
```

### Table dump (all rows)

Returns all rows from a given table (`SELECT *`). Requires DB. Useful for exporting data or when the DB is not accessible elsewhere. Allowed tables: `products`, `shopping_carts`, `shopping_cart_items`.

```
curl http://localhost:8080/tables/products
curl http://localhost:8080/tables/shopping_carts
curl http://localhost:8080/tables/shopping_cart_items
```

Response shape: `{"table": "<name>", "rows": [ {...}, ... ]}`.