# Используем официальный образ Go для сборки
FROM golang:1.23-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum для загрузки зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код и шаблоны
COPY . .
COPY templates/ ./templates/

# Компилируем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o  url-shortener main.go

# Создаем финальный образ
FROM alpine:latest

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем скомпилированный бинарник из builder
COPY --from=builder /app/url-shortener .
COPY --from=builder /app/templates ./templates/

# Открываем порт 8080
EXPOSE 8080

# Команда для запуска приложения
CMD ["./url-shortener"]
