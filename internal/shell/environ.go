package shell

import (
	"os"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// osEnvironment is the process's own environment, so that changes reach
// the programs c64sh starts.
type osEnvironment struct{}

var _ interp.Environment = osEnvironment{}

func (osEnvironment) Lookup(name string) (string, bool) { return os.LookupEnv(name) }
func (osEnvironment) Set(name, value string) error      { return os.Setenv(name, value) }
func (osEnvironment) Unset(name string) error           { return os.Unsetenv(name) }
func (osEnvironment) List() []string                    { return os.Environ() }
