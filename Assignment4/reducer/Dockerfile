FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY shared/ shared/
COPY reducer/ reducer/

# Build the reducer binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /reducer ./reducer

# --- Runtime stage ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /reducer /reducer

EXPOSE 8082
ENV PORT=8082

CMD ["/reducer"]