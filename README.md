# Инструкция по запуску и работе с Docker-compose 

Перед запуском необходимо скопировать .env.example в корне проекта и сохранить как .env:
>cp .env.example .env

Запуск контейнеров осуществляется командой:
>docker-compose up -d

Запуск миграций с хоста:
>migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5433/maindb?sslmode=disable" up

# Инструкция по добавлению нового администратора

Вход в консоль контейнера Golang:
>docker exec -it golang sh

Запуск CLI для добавления нового администратора после входа в контейнер Golang:
>go -C /app run ./cmd/cli-tools/main.go createAdmin --email=test@mail.com