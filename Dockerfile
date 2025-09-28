# Этап 1: Сборка
FROM golang:1.25-alpine AS builder

# Устанавливаем сертификаты и git
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статичный бинарник без отладочной информации для минимального размера
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bot ./cmd/bot

# Этап 2: Финальный образ
FROM scratch

# Копируем SSL сертификаты из builder, чтобы бот мог подключаться к API Telegram
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Копируем только скомпилированный бинарник
COPY --from=builder /app/bot /bot

# Запускаем бота
CMD ["/bot"]
