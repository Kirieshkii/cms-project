# Инструкция по запуску и работе с Docker-compose 

Перед запуском необходимо скопировать .env.example в корне проекта и сохранить как .env:
>cp .env.example .env

Запуск контейнеров осуществляется командой:
>docker-compose up -d


## Запуск всех тестов:

Вход в консоль контейнера Golang:
>docker exec -it golang sh

Запуск миграций главной БД из контейнера golang:
>migrate -path=/app/migrations -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" up

Запуск тестов:
>go test /app/... -v

Используемая версия mockery 2.53.5:
>go install github.com/vektra/mockery/v2@v2.53.5

Выйти из контейнера
>exit

# Инструкция по добавлению нового администратора

Вход в консоль контейнера Golang:
>docker exec -it golang sh

Запуск CLI для добавления нового администратора после входа в контейнер Golang:
>go -C /app run ./cmd/cli-tools/main.go createAdmin --email=test@mail.com

Запуск сервера
>go -C /app run ./cmd/cms-api/main.go

___

# Как проверить Redis/ключи вручную (отладка)

>После вызова /logout в контейнере (или локально, если redis контейнер работает):
>зайти в redis контейнер (или локально)
>>docker exec -it redis sh
>или подключиться с хоста
>>redis-cli -h 127.0.0.1 -p 6379 -a rpass
>
>найти ключи blacklist
>>KEYS blacklist:jti:*
>посмотреть конкретный ключ
>>GET blacklist:jti:<jti-value>   # вернёт "true" \
>>TTL blacklist:jti:<jti-value>
>
>Если ключа нет — Revoke не сработал.

---

## 🔍 Healthcheck

После запуска сервера можно проверить его состояние:

```bash
# Базовая проверка (сервер работает)
curl http://localhost:8080/health

# Проверка готовности (БД и Redis доступны)
curl http://localhost:8080/health/ready

# Проверка живучести (процесс работает)
curl http://localhost:8080/health/live
```

**Из контейнера:**
```bash
docker exec -it golang wget -qO- http://localhost:8080/health
```