package wellness_test

import (
	"testing"
	"time"

	"github.com/xsyetopz/go-mamusiabtw/internal/wellness"
)

func TestParseScheduleAndNext(t *testing.T) {
	t.Parallel()

	schedule, err := wellness.ParseSchedule(" 0 9 * * * ")
	if err != nil {
		t.Fatalf("ParseSchedule: %v", err)
	}

	if schedule.Spec() != "0 9 * * *" {
		t.Fatalf("unexpected spec: %q", schedule.Spec())
	}

	loc := time.FixedZone("EEST", 3*60*60)
	from := time.Date(2026, time.April, 2, 5, 30, 0, 0, time.UTC)
	got := schedule.Next(from, loc)
	want := time.Date(2026, time.April, 2, 6, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("unexpected next run: got %s want %s", got, want)
	}
}

func TestParseScheduleRejectsBlank(t *testing.T) {
	t.Parallel()

	if _, err := wellness.ParseSchedule(" "); err == nil {
		t.Fatalf("expected blank schedule error")
	}
}

func TestLoadLocation(t *testing.T) {
	t.Parallel()

	loc, name, err := wellness.LoadLocation("")
	if err != nil {
		t.Fatalf("LoadLocation(default): %v", err)
	}
	if loc != time.UTC || name != "UTC" {
		t.Fatalf("unexpected default timezone: %v %q", loc, name)
	}

	loc, name, err = wellness.LoadLocation("Europe/Tallinn")
	if err != nil {
		t.Fatalf("LoadLocation(valid): %v", err)
	}
	if loc == nil || name != "Europe/Tallinn" {
		t.Fatalf("unexpected timezone: %v %q", loc, name)
	}

	if _, _, err := wellness.LoadLocation("Not/AZone"); err == nil {
		t.Fatalf("expected invalid timezone error")
	}
}
