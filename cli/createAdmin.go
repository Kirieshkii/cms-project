/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Kirieshkii/cms-project/internal/db"
	"github.com/Kirieshkii/cms-project/internal/logger"
	"github.com/Kirieshkii/cms-project/internal/store/pgxstore"
	"github.com/Kirieshkii/cms-project/internal/user/service"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// createAdminCmd represents the createAdmin command
var createAdminCmd = &cobra.Command{
	Use:   "createAdmin",
	Short: "Create a new admin",
	Long:  `Create a new admin with the given email and password by flag --email=<email>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return createAdm(cmd)
	},
}

func createAdm(cmd *cobra.Command) error {
	ctx := context.Background() // создаём контекст для всей операции

	// Инициализация логгера для CLI (используем local для удобочитаемого вывода)
	log := logger.SetupLogger("local")

	email, err := cmd.Flags().GetString("email")
	if err != nil {
		return fmt.Errorf("не удается считать email: %w", err)
	}

	if email == "" {
		return fmt.Errorf("email является обязательным полем")
	}

	fmt.Print("Введите пароль: ")

	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("ошибка чтения пароля: %w", err)
	}
	password := string(pw)

	fmt.Print("\nПовторите пароль: ")

	pw, err = term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("ошибка чтения пароля: %w", err)
	}
	if password != string(pw) {
		return errors.New("пароли не совпадают")
	}

	// Инициализация пула pgxpool
	dsn, err := db.BuildDSNFromEnv()
	if err != nil {
		log.Error("ошибка сборки DSN", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, dsn) // явная проверка ошибки
	if err != nil {
		log.Error("ошибка инициализации пула БД", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Создаём хранилище
	store := pgxstore.New(pool)

	// Создаём админа через сервис с прокидыванием ctx
	if err := service.CreateAdmin(ctx, store, email, password, log); err != nil {
		return fmt.Errorf("не удается создать админа с email %s: %w", email, err)
	}

	log.Info("createAdmin успешно выполнено", "email", email)
	return nil
}

func init() {
	rootCmd.AddCommand(createAdminCmd)
	createAdminCmd.Flags().String("email", "", "Email of the new admin")
}
