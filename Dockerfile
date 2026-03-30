# ==========================================
# Stage 1: The Builder
# ==========================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy the module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire backend structure (cmd, internal, config, etc.)
COPY . .

# Build the statically linked Go binary. 
RUN CGO_ENABLED=0 GOOS=linux go build -o hinsight-api ./server

# ==========================================
# Stage 2: The Final Minimal Image
# ==========================================
FROM alpine:3.19.1

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/hinsight-api .
COPY --from=builder /app/sql/migrations ./sql/migrations

# Expose the port your Echo app listens on (adjust if you use a different port)
EXPOSE 8080

# Run the compiled binary
CMD ["./hinsight-api"]