/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/Kirieshkii/cms-project/internal/store/sqlstore"
	"github.com/Kirieshkii/cms-project/internal/user/service"
	_ "github.com/lib/pq"
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

	//host=localhost port=5433 - для запуска с хоста
	//host=postgresdb port=5432 - для запуска с контейнера Го
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка инициализации БД: %w ", err)
	}

	defer db.Close()

	/*if err = db.Ping(); err != nil {
		return fmt.Errorf("нет связи с БД: %w", err)
	}*/

	s := sqlstore.New(db)

	if err = service.CreateAdmin(s, email, password); err != nil {
		return fmt.Errorf("не удается создать админа с email: %s: %w ", email, err)
	}

	fmt.Printf("\ncreateAdmin успешно выполнено с email: %s ", email)

	return nil
}

func init() {
	rootCmd.AddCommand(createAdminCmd)

	createAdminCmd.Flags().String("email", "", "Email of the new admin")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createAdminCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createAdminCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
