package repoitory_test

import (
	"database/sql"
	"fmt"
	mathrand "math/rand"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storage "github.com/Kirieshkii/cms-project/internal/store"
	"github.com/Kirieshkii/cms-project/internal/store/sqlstore"
	"github.com/Kirieshkii/cms-project/internal/user/model"
)

const (
// testDBname = "testdb"
)

func TestCreate(t *testing.T) {
	//testcase
	u := &model.User{
		Email:             RandEmail(),
		EncryptedPassword: "encryptedpassword",
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))

	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = db.Close()
	})

	err = db.Ping()
	require.NoError(t, err)

	//Create
	s := sqlstore.New(db)

	err = s.User().Create(u)
	assert.NoError(t, err)

	err = s.User().Create(u)
	assert.ErrorIs(t, err, storage.ErrUserAlreadyExists)

}

func RandEmail() string {
	n := mathrand.Intn(10000)
	return fmt.Sprintf("test%d@gmail.com", n)
}
