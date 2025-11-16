package tests

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/suite"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/store/pgxstore"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

type UserRepositoryTestSuite struct {
	DBTestSuite
	Store storage.Store
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}

func (s *UserRepositoryTestSuite) SetupSuite() {
	// Инициализация пула и запуск миграций
	s.DBTestSuite.SetupSuite()

	// Инициализация Store
	s.Store = pgxstore.New(s.Pool)

	fmt.Println("✅ Store и пул подключений успешно инициализированы")
}

func (s *UserRepositoryTestSuite) SetupTest() {
	// Очищаем таблицу users перед каждым тестом
	err := CleanupTables(context.Background(), s.Pool, "users")
	if err != nil {
		s.T().Fatalf("❌ Не удалось очистить таблицу users: %v", err)
	}

	fmt.Println("🧹 Таблица users очищена перед тестом")
}

func (s *UserRepositoryTestSuite) TestCreateUser() {
	ctx := context.Background()
	u := &model.User{
		Email:             RandEmail(),
		EncryptedPassword: "encryptedpassword",
	}

	fmt.Printf("🔹 Создаем пользователя с email: %s\n", u.Email)
	err := s.Store.User().Create(ctx, u)
	s.Require().NoError(err, "❌ Ошибка при создании пользователя")

	// Получаем версию токена
	version, err := s.Store.User().GetTokenVersion(ctx, u.ID)
	s.Require().NoError(err, "❌ Ошибка при получении версии токена")
	s.Assert().Equal(1, version, "❌ Неверная версия токена по умолчанию")

	fmt.Println("✅ Пользователь успешно создан, версия токена проверена")

	// Повторное создание → должно вернуть ErrUserAlreadyExists
	fmt.Printf("🔹 Пытаемся создать пользователя с тем же email: %s\n", u.Email)
	err = s.Store.User().Create(ctx, u)
	s.Require().ErrorIs(err, storage.ErrUserAlreadyExists, "❌ Повторное создание пользователя не вернуло ожидаемую ошибку")

	fmt.Println("✅ Проверка дубликата пользователя прошла успешно")
}

// Генерация случайного email
func RandEmail() string {
	n := rand.Intn(10000)
	return fmt.Sprintf("test%d@gmail.com", n)
}
