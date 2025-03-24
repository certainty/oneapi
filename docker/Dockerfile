FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o oneapi ./cmd/oneapi/main.go

# Deployment stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /app/oneapi .
CMD ["/app/oneapi", "/app/manifest.yaml"]
