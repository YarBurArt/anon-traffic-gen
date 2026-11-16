# build stage: cgo off keeps the result a static binary, so the runtime image
# needs nothing but certificates
FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/spoof-http ./cmd/spoof-http

FROM alpine:3.22

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=build /out/spoof-http .
COPY config.yaml .

CMD ["/app/spoof-http", "--config=/app/config.yaml"]
