# First stage: Build the Go binary
FROM golang:1.24.1-alpine AS builder

# Add Maintainer Info
LABEL maintainer="Prokopis Antoniadis prokopis123@gmail.com"

# set initial working directory
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod tidy -v && go mod verify

COPY . .

# Set environment variables for Go build
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GOARCH=amd64

# Build the Go app
RUN go build -a -installsuffix cgo -o main ./main.go
# RUN chmod -R 777 /app/fiber.log

###############Application Image################
# Second stage: Run the binary in a minimal image
# Run stage
FROM alpine:latest as release


WORKDIR /app

# Copy the binary from the builder stage
# COPY --from=builder /app/.env .env
COPY --from=builder /app/main /app/main

# Add packages
RUN apk -U upgrade \
    && apk add --no-cache dumb-init ca-certificates \
    && chmod +x /app/main

# EXPOSE PORT 8080
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD [ "./main" ]
