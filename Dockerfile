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
FROM golang:1.25-alpine AS backend-builder
WORKDIR /src
# Install git and certificates
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy the compiled frontend assets from the previous stage to the location Go expects
COPY --from=frontend-builder /app/dist ./pkg/ui/dist
# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/gateway ./cmd/gateway

# Stage 3: Final Runtime Image
FROM alpine:latest
WORKDIR /app
# Copy only the binary and CA certificates
COPY --from=backend-builder /bin/gateway /bin/gateway
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Expose ports
EXPOSE 8899 2100
# Run
ENTRYPOINT ["/bin/gateway"]
