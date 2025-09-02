# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o devproxy ./cmd/devproxy

# Final stage - minimal scratch image
FROM scratch

# Copy the statically linked binary
COPY --from=builder /app/devproxy /devproxy

# Run the binary
ENTRYPOINT ["/devproxy"]