AWS_REGION   := us-east-1
AWS_ACCOUNT  := $(shell aws sts get-caller-identity --query Account --output text)
ECR_URL      := $(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com/a10-kv-node
IMAGE_TAG    := latest

# ── Local Testing ─────────────────────────────────────────────────────────────

# Test W=5 R=1 locally
local-w5r1:
	docker compose --profile leader-w5r1 up --build

# Test W=1 R=5 locally
local-w1r5:
	docker compose --profile leader-w1r5 up --build

# Test W=3 R=3 locally
local-w3r3:
	docker compose --profile leader-w3r3 up --build

# Test leaderless locally
local-leaderless:
	docker compose --profile leaderless up --build

# ── AWS Deployment ────────────────────────────────────────────────────────────

# Build and push Docker image to ECR
build-push:
	cd node && \
	docker build -t $(ECR_URL):$(IMAGE_TAG) . && \
	aws ecr get-login-password --region $(AWS_REGION) | \
		docker login --username AWS --password-stdin $(ECR_URL) && \
	docker push $(ECR_URL):$(IMAGE_TAG)

# Deploy W=5 R=1 leader-follower to AWS
deploy-w5r1: build-push
	cd terraform && \
	terraform init && \
	terraform apply -auto-approve \
		-var="image_url=$(ECR_URL):$(IMAGE_TAG)" \
		-var="w=5" \
		-var="r=1" \
		-var="mode=leader-follower"

# Deploy W=1 R=5 leader-follower to AWS
deploy-w1r5: build-push
	cd terraform && \
	terraform apply -auto-approve \
		-var="image_url=$(ECR_URL):$(IMAGE_TAG)" \
		-var="w=1" \
		-var="r=5" \
		-var="mode=leader-follower"

# Deploy W=3 R=3 quorum to AWS
deploy-w3r3: build-push
	cd terraform && \
	terraform apply -auto-approve \
		-var="image_url=$(ECR_URL):$(IMAGE_TAG)" \
		-var="w=3" \
		-var="r=3" \
		-var="mode=leader-follower"

# Deploy leaderless to AWS
deploy-leaderless: build-push
	cd terraform && \
	terraform apply -auto-approve \
		-var="image_url=$(ECR_URL):$(IMAGE_TAG)" \
		-var="w=5" \
		-var="r=1" \
		-var="mode=leaderless"

# Destroy all AWS resources
destroy:
	cd terraform && terraform destroy -auto-approve

# ── Load Testing ──────────────────────────────────────────────────────────────
# Usage: make loadtest LEADER=http://your-alb-dns WRITE_RATIO=0.5 CONFIG=W5R1

loadtest:
	cd loadtest && \
	LEADER_ADDR=$(LEADER) \
	WRITE_RATIO=$(WRITE_RATIO) \
	CONFIG=$(CONFIG) \
	locust -f locustfile.py --headless -u 20 -r 2 -t 60s \
		--host $(LEADER) \
		--csv=results/$(CONFIG)-$(WRITE_RATIO)

# ── Unit Tests ────────────────────────────────────────────────────────────────
test:
	cd tests && \
	LEADER_ADDR=$(LEADER) \
	FOLLOWER1_ADDR=$(FOLLOWER1) \
	FOLLOWER2_ADDR=$(FOLLOWER2) \
	go test -v ./...