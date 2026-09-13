FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/telegram-user-info-bot ./cmd/bot

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S bot \
    && adduser -S -G bot bot

COPY --from=builder /out/telegram-user-info-bot /usr/local/bin/telegram-user-info-bot

USER bot

ENTRYPOINT ["/usr/local/bin/telegram-user-info-bot"]
