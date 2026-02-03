# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app
# Copy package files first to leverage cache
COPY webapp/package*.json ./
RUN npm install
# Copy source and build
COPY webapp/ .
RUN npm run build

# Stage 2: Build Backend (with embedded frontend)
FROM golang:1.24-bullseye AS backend-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy the compiled frontend assets from the previous stage to the location Go expects
COPY --from=frontend-builder /app/dist ./pkg/ui/dist
# Build the binary with low concurrency (-p 1) to avoid OOM in CI, and verbose output
RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -x -v -o /bin/gateway ./cmd/gateway

# Stage 3: Final Runtime Image
FROM alpine:latest
WORKDIR /app
# Copy only the binary
COPY --from=backend-builder /bin/gateway /bin/gateway
# Expose ports
EXPOSE 8899 2100
# Run
CMD ["/bin/gateway"]