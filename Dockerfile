FROM golang:1.25-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/app ./cmd/app \
 && CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/cli ./cmd/cli

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=builder /out/app /usr/local/bin/app
COPY --from=builder /out/cli /usr/local/bin/cli
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/app"]
