FROM golang:1.23

# Install system dependencies including Chromium
RUN apt-get update && apt-get install -y \
    wget \
    curl \
    unzip \
    git \
    build-essential \
    ca-certificates \
    libglib2.0-0 \
    libnss3 \
    libgconf-2-4 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    fonts-liberation \
    lsb-release \
    xdg-utils \
    gnupg \
    chromium \
    && rm -rf /var/lib/apt/lists/*

# Set environment variables for chromedp
ENV CHROME_PATH=/usr/bin/chromium
ENV CHROMIUM_USER_DATA_DIR=/tmp/chromium

WORKDIR /app

COPY . .

RUN go mod tidy && go build -o mangaroo cmd/main.go

CMD ["./mangaroo"]