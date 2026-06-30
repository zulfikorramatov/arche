package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zulfikorramatov/arche/cmd/cli/deps"
	"github.com/zulfikorramatov/arche/cmd/cli/user"
	"github.com/zulfikorramatov/arche/internal/config"
	"github.com/zulfikorramatov/arche/pkg/postgres"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	d := &deps.Deps{}

	root := &cobra.Command{
		Use:   "arche-cli",
		Short: "CLI tools for arche",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			d.Pool, err = postgres.New(cmd.Context(), postgres.Config(cfg.Postgres))
			if err != nil {
				return fmt.Errorf("connect postgres: %w", err)
			}

			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if d.Pool != nil {
				d.Pool.Close()
			}
		},
	}

	root.SetContext(context.Background())
	root.AddCommand(user.NewUserCmd(d))

	return root
}
