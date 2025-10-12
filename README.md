# Инструкция по запуску и работе с Docker-compose 

Перед запуском необходимо скопировать .env.example в корне проекта и сохранить как .env:
>cp .env.example .env

Запуск контейнеров осуществляется командой:
>docker-compose up -d

Запуск миграций главной БД с хоста:
>migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5433/${POSTGRES_DB}?sslmode=disable" up

Запуск миграций тестовой БД с хоста:
>migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5433/testdb?sslmode=disable" up

## Запуск интеграционных тестов с хоста c установленными переменными окружения:
### Тест 1
POSTGRES_USER=petproject \
POSTGRES_PASSWORD=hardpass \
POSTGRES_DB=testdb \
POSTGRES_HOST=localhost \
POSTGRES_PORT=5433 \
go test -v ./tests/integration -run ^TestCreate$

### Тест 2
go test -v ./internal/user/service -run ^TestCreateAdmin$

Используемая версия mockery 2.53.5:
>go install github.com/vektra/mockery/v2@v2.53.5


# Инструкция по добавлению нового администратора

Вход в консоль контейнера Golang:
>docker exec -it golang sh

Запуск CLI для добавления нового администратора после входа в контейнер Golang:
>go -C /app run ./cmd/cli-tools/main.go createAdmin --email=test@mail.com