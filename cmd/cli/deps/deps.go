package deps

import (
	"github.com/zulfikorramatov/arche/pkg/postgres"
)

type Deps struct {
	Pool *postgres.Pool
}
