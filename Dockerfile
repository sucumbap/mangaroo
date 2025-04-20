FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mangaroo ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/mangaroo .
COPY --from=builder /app/configs ./configs

# Install Chromium and dependencies
RUN apk add --no-cache \
    chromium \
    harfbuzz \
    nss \
    freetype \
    ttf-freefont \
    && rm -rf /var/cache/apk/*

ENV CHROME_PATH=/usr/bin/chromium-browser
ENV CHROMIUM_USER_DATA_DIR=/tmp/chromium

EXPOSE 8080

CMD ["./mangaroo"]