package user

import (
	"github.com/spf13/cobra"

	"github.com/zulfikorramatov/arche/cmd/cli/deps"
)

func NewUserCmd(d *deps.Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}

	cmd.AddCommand(newCreateCmd(d))
	cmd.AddCommand(newDeleteCmd(d))

	return cmd
}
