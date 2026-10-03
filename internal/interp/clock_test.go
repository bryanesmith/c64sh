package interp

import (
	"testing"
	"time"

	"github.com/bryanesmith/c64sh/internal/basicerr"
)

// clockSession runs lines at successive clock times: before line i, the
// clock reads start plus at[i].
func clockSession(t *testing.T, at []time.Duration, ls ...string) (string, error) {
	t.Helper()
	start := time.Unix(1000, 0)
	now := start
	rec := &recorder{}
	in := New(rec)
	in.SetClock(func() time.Time { return now })
	var err error
	for i, l := range ls {
		now = start.Add(at[i])
		err = enter(in, l)
	}
	return rec.String(), err
}

// @spec INTERP-143
func TestTI(t *testing.T) {
	out, err := clockSession(t, []time.Duration{0, time.Second, 90 * time.Second},
		"PRINT TI", "PRINT TI", "PRINT TI")
	if err != nil || out != " 0 \n 60 \n 5400 \n" {
		t.Errorf("TI: %q, %v", out, err)
	}
	out, _ = clockSession(t, []time.Duration{0, 2 * time.Second}, `TI$="235959"`, "PRINT TI")
	if out != " 60 \n" {
		t.Errorf("TI after 24 hours: %q, want it wrapped", out)
	}
	out, _ = clockSession(t, []time.Duration{time.Second / 120}, "PRINT TI")
	if out != " 0 \n" {
		t.Errorf("TI rounds down: %q", out)
	}
}

// @spec INTERP-144
func TestTIString(t *testing.T) {
	out, _ := clockSession(t, []time.Duration{0, time.Hour + 2*time.Minute + 3*time.Second},
		"PRINT TI$", "PRINT TI$")
	if out != "000000\n010203\n" {
		t.Errorf("TI$: %q", out)
	}
}

// @spec INTERP-145
func TestSetTIString(t *testing.T) {
	out, err := clockSession(t, []time.Duration{0, 0, 10 * time.Second},
		`TI$="123000"`, "PRINT TI;TI$", "PRINT TI$")
	if err != nil || out != " 2700000 123000\n123010\n" {
		t.Errorf("set TI$: %q, %v", out, err)
	}
	for _, bad := range []string{`TI$="12300"`, `TI$="1230000"`, `TI$="12A000"`, `TI$=""`} {
		if _, err := clockSession(t, []time.Duration{0}, bad); !isKind(err, basicerr.IllegalQuantity) {
			t.Errorf("%s: %v, want ILLEGAL QUANTITY", bad, err)
		}
	}
	if _, err := clockSession(t, []time.Duration{0}, `TI$=5`); !isKind(err, basicerr.TypeMismatch) {
		t.Errorf("TI$=5: %v, want TYPE MISMATCH", err)
	}
}
