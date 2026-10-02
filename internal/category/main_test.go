package category

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		err := godotenv.Load("../../.env.test")
		if err != nil {
			//nolint:revive // Setup failure must terminate before m.Run.
			os.Exit(1)
		}
	}

	m.Run()
}
