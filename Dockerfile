# JUST SKETCH, WITHOUT DEBAG not to use.

# ubuntu latest as builder base because dependencies
FROM ubuntu AS builder

WORKDIR /app

# Copy go.mod and go.sum to cache dependencies
COPY go.mod go.sum config.yaml ./
# Install go 
RUN apt-get update && apt-get upgrade -y 
RUN apt-get install -y golang
RUN apt-get install -y ca-certificates
#RUN go mod download
RUN go build -v

# Use a lighter image for running
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates

WORKDIR /app

# no expose because we only download and fetch

COPY --from=builder /app/spoof-http .
COPY --from=builder /app/config.yaml .

CMD ["./spoof-http", "--config=config.yaml"]
