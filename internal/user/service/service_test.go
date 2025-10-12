package service_test

import (
	"testing"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/store/mocks"
	"github.com/Kirieshkii/cms-project/internal/user/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateAdmin(t *testing.T) {
	// test cases
	testcases := []struct {
		name     string
		email    string
		password string
		expErr   bool
		mockErr  error
	}{
		{
			name:     "Valid",
			email:    "test1@gmail.com",
			password: "reallyhardpass",
			expErr:   false,
			mockErr:  nil,
		},
		{
			name:     "Invalid, empty email",
			email:    "",
			password: "reallyhardpass",
			expErr:   true,
			mockErr:  nil,
		},
		{
			name:     "Invalid, wrong email",
			email:    "bibabobamail.com",
			password: "reallyhardpass",
			expErr:   true,
			mockErr:  nil,
		},
		{
			name:     "Invalid, empty password",
			email:    "test1@gmail.com",
			password: "       ",
			expErr:   true,
			mockErr:  nil,
		},
		{
			name:     "Invalid, short password",
			email:    "test1@gmail.com",
			password: "short",
			expErr:   true,
			mockErr:  nil,
		},
		{
			name:     "Invalid, user is exists",
			email:    "test1@gmail.com",
			password: "reallyhardpass",
			expErr:   true,
			mockErr:  storage.ErrUserAlreadyExists,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			UserRepoMock := mocks.NewUserRepository(t)
			storeMock := &mocks.Store{
				UserRepo: UserRepoMock,
			}

			if !(tc.expErr && tc.mockErr == nil) {
				UserRepoMock.On("Create", mock.Anything).
					Return(tc.mockErr).Once()
			}

			err := service.CreateAdmin(storeMock, tc.email, tc.password)

			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tc.mockErr == storage.ErrUserAlreadyExists {
				assert.ErrorIs(t, err, storage.ErrUserAlreadyExists)
			}

			UserRepoMock.AssertExpectations(t)
		})

	}
}
