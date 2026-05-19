FROM golang:1.26-alpine3.23 as builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/*.go .

RUN CGO_ENABLED=0 go build -o b2bua .

FROM alpine:3.23.4

RUN apk add --no-cache \
    bash \
    curl \
    ca-certificates \
    iproute2 \
    net-tools \
    openssl \
    sngrep

RUN addgroup -S b2bua && adduser -S b2bua -G b2bua

WORKDIR /app

COPY entrypoint.sh .

COPY --from=builder /app/b2bua .

RUN chmod +x /app/b2bua /app/entrypoint.sh \
    && chown -R b2bua:b2bua /app

USER b2bua

ENTRYPOINT ["./entrypoint.sh"]
CMD ["start"]
