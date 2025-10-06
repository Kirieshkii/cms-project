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
		return fmt.Errorf("can't get email: %w", err)
	}

	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("Введите пароль: ")

	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("ошибка чтения пароля: %w", err)
	}

	password := string(pw)

	fmt.Println("\nПовторите пароль: ")

	pw, err = term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("ошибка чтения пароля: %w", err)
	}

	if password != string(pw) {
		return errors.New("пароли не совпадают")
	}

	connStr := fmt.Sprintf(
		"host=postgresdb port=5432 user=%s password=%s dbname=%s sslmode=disable",
		//TODO: read toml config
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("ошибка инициализации БД: %w", err)
	}

	defer db.Close()

	s := sqlstore.New(db)

	if err = service.CreateAdmin(s, email, password); err != nil {
		return fmt.Errorf("can't create user with email %s: %w", email, err)
	}

	fmt.Printf("createAdmin successfully done with email: %s", email)

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
