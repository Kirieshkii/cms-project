package logger

import (
	"log/slog"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

// setupLogger зависит от входного параметра окружения env, т.к. локально мы хотим видеть текстовые логи,
// на сервере dev или prod хотим видеть JSON, на dev уровня debug, на prod не ниже INFO
func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog() // только для локального использования (в прод такое нельзя)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		// По умолчанию используем dev режим, если окружение не указано
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

// setupPrettySlog создаёт логгер с текстовым форматом для локального использования.
// Только для локального использования (в прод такое нельзя).
func setupPrettySlog() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // В локальном окружении показываем все логи включая debug
		// AddSource: true, // Можно включить для показа файла и строки, но это замедляет
	}

	// Text handler для удобочитаемого вывода в консоли
	handler := slog.NewTextHandler(os.Stdout, opts)

	return slog.New(handler)
}

// New создаёт новый slog.Logger с выводом в stdout в JSON формате.
// Для production рекомендуется использовать JSON формат для удобства парсинга.
// Deprecated: Используйте SetupLogger вместо этой функции для правильной настройки по окружению.
func New() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Можно сделать конфигурируемым через env
	}

	// JSON handler для структурированного логирования
	handler := slog.NewJSONHandler(os.Stdout, opts)

	return slog.New(handler)
}

// NewText создаёт новый slog.Logger с выводом в stdout в текстовом формате.
// Удобен для разработки, так как более читаемый.
// Deprecated: Используйте SetupLogger вместо этой функции для правильной настройки по окружению.
func NewText() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	// Text handler для удобочитаемого вывода
	handler := slog.NewTextHandler(os.Stdout, opts)

	return slog.New(handler)
}
