# 1. Базовый образ Go
FROM golang:1.25.1-alpine3.22

# 1.1 Устанавливаем git и ca-certificates
RUN apk add --no-cache git ca-certificates


# 2. Устанавливаем migrate
# Используем опубликованный релиз (v4.17.1 — актуальная стабильная версия на момент написания)
ARG MIGRATE_VERSION=v4.17.1
# Используем публичный прокси Go для более стабильной загрузки модулей
ENV GOPROXY=https://proxy.golang.org,direct
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$MIGRATE_VERSION

# 2.1 Добавляем go/bin в PATH
ENV PATH=$PATH:/go/bin

# 3. Контейнер стартует в shell для интерактивной работы
CMD ["sh", "-c", "sleep infinity"]
