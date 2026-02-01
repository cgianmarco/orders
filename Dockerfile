# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod ./
RUN go mod download
COPY . ./
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/orders ./...

# Runtime stage (small)
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/orders /orders
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/orders"]