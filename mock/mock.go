package mock

import (
	_ "embed"
)

//go:embed seed.sql
var SeedSQL string
