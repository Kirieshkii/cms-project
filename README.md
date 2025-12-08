# Инструкция по запуску и работе с Docker-compose 

Перед запуском необходимо скопировать .env.example в корне проекта и сохранить как .env:
>cp .env.example .env

Запуск контейнеров осуществляется командой:
>docker-compose up -d

Запуск миграций главной БД с хоста:
>migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5433/${POSTGRES_DB}?sslmode=disable" up

## Запуск всех тестов:

Вход в консоль контейнера Golang:
>docker exec -it golang sh
Запуск тестов:
>go test /app/... -v

Используемая версия mockery 2.53.5:
>go install github.com/vektra/mockery/v2@v2.53.5


# Инструкция по добавлению нового администратора

Вход в консоль контейнера Golang:
>docker exec -it golang sh

Запуск CLI для добавления нового администратора после входа в контейнер Golang:
>go -C /app run ./cmd/cli-tools/main.go createAdmin --email=test@mail.com