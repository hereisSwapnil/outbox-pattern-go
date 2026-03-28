FROM golang:alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binaries
RUN go build -o /bin/app ./cmd/app
RUN go build -o /bin/relay ./cmd/relay

# Final image
FROM alpine:latest

# Certificates required for HTTPS requests often needed in relay
RUN apk --no-cache add ca-certificates

COPY --from=builder /bin/app /bin/app
COPY --from=builder /bin/relay /bin/relay

EXPOSE 8080

CMD ["/bin/app"]
