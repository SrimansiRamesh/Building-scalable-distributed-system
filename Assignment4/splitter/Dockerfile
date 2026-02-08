FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY shared/ shared/
COPY splitter/ splitter/

# Build the splitter binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /splitter ./splitter

# --- Runtime stage ---
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /splitter /splitter

EXPOSE 8080
ENV PORT=8080

CMD ["/splitter"]