package shell

import (
	"os"
	"strings"
	"testing"

	"github.com/bryanesmith/c64sh/internal/interp"
)

// @spec SHELL-ENV-001
func TestRunUsesConfigEnvironment(t *testing.T) {
	env := interp.MapEnvironment{"HOME": "/home/c64"}
	stdout, _ := runStyled(t, Config{Env: env}, "PRINT ENVIRON$(\"HOME\")\nENVIRON \"SET=1\"\n")
	if stdout != "/home/c64\n" || env["SET"] != "1" {
		t.Errorf("stdout %q, env %v: want the configured environment read and set", stdout, env)
	}
}

// @spec SHELL-ENV-001
func TestOSEnvironment(t *testing.T) {
	t.Setenv("C64SH_TEST_VAR", "one")
	var env osEnvironment
	if v, ok := env.Lookup("C64SH_TEST_VAR"); !ok || v != "one" {
		t.Errorf("Lookup = %q, %v", v, ok)
	}
	if err := env.Set("C64SH_TEST_VAR", "two"); err != nil || os.Getenv("C64SH_TEST_VAR") != "two" {
		t.Errorf("Set: %v, now %q", err, os.Getenv("C64SH_TEST_VAR"))
	}
	found := false
	for _, kv := range env.List() {
		found = found || strings.HasPrefix(kv, "C64SH_TEST_VAR=two")
	}
	if !found {
		t.Error("List lacks C64SH_TEST_VAR")
	}
	if err := env.Unset("C64SH_TEST_VAR"); err != nil {
		t.Fatal(err)
	}
	if _, ok := os.LookupEnv("C64SH_TEST_VAR"); ok {
		t.Error("Unset left the variable set")
	}
}
