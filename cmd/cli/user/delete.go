package user

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/zulfikorramatov/arche/cmd/cli/deps"
)

func newDeleteCmd(d *deps.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user by ID or username",
		RunE: func(cmd *cobra.Command, args []string) error {
			idStr, _ := cmd.Flags().GetString("id")
			username, _ := cmd.Flags().GetString("username")

			if idStr == "" && username == "" {
				return fmt.Errorf("provide --id or --username")
			}

			var (
				q   string
				arg any
			)

			if idStr != "" {
				parsed, err := uuid.Parse(idStr)
				if err != nil {
					return fmt.Errorf("invalid uuid: %w", err)
				}
				q = `DELETE FROM users WHERE id = $1`
				arg = parsed
			} else {
				q = `DELETE FROM users WHERE username = $1`
				arg = username
			}

			result, err := d.Pool.Exec(cmd.Context(), q, arg)
			if err != nil {
				return fmt.Errorf("delete user: %w", err)
			}

			if result.RowsAffected() == 0 {
				fmt.Println("user not found")
				return nil
			}

			fmt.Println("user deleted")
			return nil
		},
	}

	cmd.Flags().String("id", "", "user UUID")
	cmd.Flags().String("username", "", "username")

	return cmd
}
