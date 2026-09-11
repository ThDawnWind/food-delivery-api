package product

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		if err := godotenv.Load("../../.env.test"); err != nil {
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}
