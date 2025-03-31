# First stage: Build the Go binary
FROM golang:1.23.0-alpine AS builder

# set initial working directory
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod ./

COPY go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && go mod tidy -v && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main ./main.go
# RUN chmod -R 777 /app/fiber.log

###############Application Image################
# Second stage: Run the binary in a minimal image
# Run stage
FROM alpine:latest as release


WORKDIR /app
# Copy the binary from the builder stage
COPY --from=builder /app/.env .
COPY --from=builder /app/libnet-d76db-949683c2222d.json .

RUN apk -U upgrade \
    && apk add --no-cache dumb-init ca-certificates \
    && chmod +x /app/main

CMD ["./deploy4scrap"]