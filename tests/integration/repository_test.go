package tests

import (
	"context"
	"fmt"
	mathrand "math/rand"
	"testing"

	"github.com/stretchr/testify/suite"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/store/pgxstore"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

type UserRepositoryTestSuite struct {
	DBTestSuite
}

func TestUserRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UserRepositoryTestSuite))
}

func (s *UserRepositoryTestSuite) SetupTest() {
	// Очищаем таблицу users перед каждым тестом
	_, err := s.Pool.Exec(context.Background(), `DELETE FROM users`)
	s.Require().NoError(err)
}

func (s *UserRepositoryTestSuite) TestCreateUser() {
	repo := pgxstore.New(s.Pool)

	u := &model.User{
		Email:             RandEmail(),
		EncryptedPassword: "encryptedpassword",
	}

	// 1. Создаём пользователя
	err := repo.User().Create(context.Background(), u)
	s.Assert().NoError(err)

	// 2. Повторное создание того же пользователя → ошибка
	err = repo.User().Create(context.Background(), u)
	s.Assert().ErrorIs(err, storage.ErrUserAlreadyExists)
}

func RandEmail() string {
	n := mathrand.Intn(10000)
	return fmt.Sprintf("test%d@gmail.com", n)
}
