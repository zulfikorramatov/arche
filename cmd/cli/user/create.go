package user

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"github.com/zulfikorramatov/arche/cmd/cli/deps"
)

func newCreateCmd(d *deps.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")

			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			const check = `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
			var exists bool
			if err = d.Pool.QueryRow(cmd.Context(), check, username).Scan(&exists); err != nil {
				return fmt.Errorf("check user: %w", err)
			}
			if exists {
				return fmt.Errorf("user %q already exists", username)
			}

			id := uuid.New()
			now := time.Now().UTC()

			const q = `INSERT INTO users (id, username, password, created_at) VALUES ($1, $2, $3, $4)`
			if _, err = d.Pool.Exec(cmd.Context(), q, id, username, string(hash), now); err != nil {
				return fmt.Errorf("insert user: %w", err)
			}

			fmt.Printf("user created: id=%s username=%s\n", id, username)
			return nil
		},
	}

	cmd.Flags().String("username", "", "username (required)")
	cmd.Flags().String("password", "", "password (required)")
	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}
