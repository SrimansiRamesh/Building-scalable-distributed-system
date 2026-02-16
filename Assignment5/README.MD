# Product API — Assignment 5

A simple in-memory Product API built with Go and Gin, implementing the Product
endpoints from the E-commerce OpenAPI specification. Deployed to AWS ECS Fargate
using Terraform for infrastructure-as-code.

## Repository Structure

```
.
├── src/                        # Application source code
│   ├── main.go                 # Entry point, Gin router setup
│   ├── handler.go              # HTTP handlers (GET, POST) and input validation
│   ├── model.go                # Data models (Product, ErrorResponse)
│   ├── store.go                # Thread-safe in-memory hashmap store (sync.RWMutex)
│   ├── go.mod                  # Go module definition
│   ├── go.sum                  # Dependency checksums
│   └── Dockerfile              # Multi-stage Docker build (golang:alpine → alpine)
├── terraform/                  # Infrastructure-as-code (Terraform)
│   ├── main.tf                 # Root module — wires together ECR, ECS, network, logging
│   ├── provider.tf             # AWS provider configuration
│   ├── variables.tf            # Input variables (region, service name, ports, etc.)
│   ├── outputs.tf              # Terraform outputs (cluster name, service name)
│   └── modules/
│       ├── ecr/                # ECR repository for Docker images
│       ├── ecs/                # ECS cluster, task definition, and Fargate service
│       ├── network/            # VPC, subnets, and security group (port 8080)
│       └── logging/            # CloudWatch log group
├── locust-tests/               # Load testing
│   └── locustfile.py           # Locust test file (HttpUser + FastHttpUser)
├── api.yaml                    # OpenAPI 3.0 specification
└── README.md                   # This file
```

## Prerequisites

- [Go](https://go.dev/dl/) (1.21+)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- [AWS CLI](https://aws.amazon.com/cli/) configured with valid credentials
- [Terraform](https://developer.hashicorp.com/terraform/downloads) (1.5+)
- [Locust](https://locust.io/) (`pip install locust`) for load testing

## API Endpoints

| Method | Path                            | Description                | Success Code |
| ------ | ------------------------------- | -------------------------- | ------------ |
| GET    | `/products/{productId}`         | Get product by ID          | 200          |
| POST   | `/products/{productId}/details` | Add/update product details | 204          |

## Running Locally

```bash
cd src
go mod tidy
go run .
```

The server starts on port 8080.

## Building and Running with Docker

```bash
cd src
go mod tidy                     # generates go.sum (required by Dockerfile)
docker build -t product-api .
docker run -p 8080:8080 product-api
```

Note: If you are on Apple Silicon (M1/M2/M3), build for the correct platform
when deploying to AWS (Fargate runs linux/amd64):

```bash
docker build --platform linux/amd64 -t product-api .
```

## Deploying to AWS with Terraform

### 1. Initialize and apply infrastructure

```bash
cd terraform
terraform init
terraform plan          # review what will be created
terraform apply         # type "yes" to confirm
```

This creates: ECR repository, ECS cluster + Fargate service, security group
(allows inbound on port 8080), and CloudWatch log group.

### 2. Push Docker image to ECR

```bash
# Get ECR repository URL
ECR_URL=$(aws ecr describe-repositories --region us-east-1 \
  --query 'repositories[?repositoryName==`ecr_service`].repositoryUri' --output text)

# Login to ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin $ECR_URL

# Build for linux/amd64 (required for Fargate)
cd ../src
docker build --platform linux/amd64 -t product-api .

# Tag and push
docker tag product-api:latest $ECR_URL:latest
docker push $ECR_URL:latest
```

### 3. Deploy to ECS

```bash
aws ecs update-service --cluster CS6650L2-cluster --service CS6650L2 \
  --force-new-deployment --region us-east-1
```

### 4. Get the public IP of the running task

```bash
TASK_ARN=$(aws ecs list-tasks --cluster CS6650L2-cluster --region us-east-1 \
  --query 'taskArns[0]' --output text)

ENI=$(aws ecs describe-tasks --cluster CS6650L2-cluster --tasks $TASK_ARN --region us-east-1 \
  --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text)

aws ec2 describe-network-interfaces --network-interface-ids $ENI --region us-east-1 \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text
```

### 5. Tear down infrastructure

```bash
cd terraform
terraform destroy
```

## API Examples and Response Codes

Replace `<host>` with `localhost` (local) or your ECS public IP (deployed).

### 204 No Content — Create a product (success)

```bash
curl -v -X POST http://:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1,
    "sku": "ABC-123-XYZ",
    "manufacturer": "Acme Corporation",
    "category_id": 456,
    "weight": 1250,
    "some_other_id": 789
  }'
```

Response: empty body, status `204 No Content`.

### 200 OK — Get an existing product

```bash
curl http://:8080/products/1
```

Response:

```json
{
  "product_id": 1,
  "sku": "ABC-123-XYZ",
  "manufacturer": "Acme Corporation",
  "category_id": 456,
  "weight": 1250,
  "some_other_id": 789
}
```

### 404 Not Found — Get a product that doesn't exist

```bash
curl http://:8080/products/999
```

Response:

```json
{
  "error": "NOT_FOUND",
  "message": "Product not found",
  "details": "No product found with ID 999"
}
```

### 400 Bad Request — Missing or invalid required fields

```bash
curl -X POST http://:8080/products/2/details \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 2,
    "sku": "",
    "manufacturer": "",
    "category_id": 0,
    "weight": 100,
    "some_other_id": 0
  }'
```

Response:

```json
{
  "error": "INVALID_INPUT",
  "message": "Validation failed",
  "details": "sku is required; manufacturer is required; category_id must be at least 1; some_other_id must be at least 1"
}
```

### 400 Bad Request — Product ID mismatch (path vs body)

```bash
curl -X POST http://:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 99,
    "sku": "SKU-001",
    "manufacturer": "Test Corp",
    "category_id": 1,
    "weight": 500,
    "some_other_id": 1
  }'
```

Response:

```json
{
  "error": "INVALID_INPUT",
  "message": "Product ID mismatch",
  "details": "Product ID in URL does not match product_id in request body"
}
```

### 400 Bad Request — Malformed JSON body

```bash
curl -X POST http://:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d 'not json'
```

Response:

```json
{
  "error": "INVALID_INPUT",
  "message": "Invalid request body",
  "details": "invalid character 'o' in literal null (expecting 'u')"
}
```

## Load Testing with Locust

```bash
pip install locust
cd locust-tests

# Run with HttpUser (standard)
locust -f locustfile.py --host http://:8080 ProductHttpUser

# Run with FastHttpUser (lightweight)
locust -f locustfile.py --host http://:8080 ProductFastHttpUser
```

Open http://localhost:8089 in your browser to start the test. Suggested parameters:

| Test     | Users | Spawn Rate | Duration |
| -------- | ----- | ---------- | -------- |
| Baseline | 10    | 2          | 60s      |
| Medium   | 100   | 10         | 60s      |
| Stress   | 500   | 50         | 60s      |
| Break it | 1000  | 100        | 60s      |
