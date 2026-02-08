FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY shared/ shared/
COPY mapper/ mapper/

# Build the mapper binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /mapper ./mapper

# --- Runtime stage ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /mapper /mapper

EXPOSE 8081
ENV PORT=8081

CMD ["/mapper"]