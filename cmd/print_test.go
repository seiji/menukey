package cmd

import (
	"strings"
	"testing"

	"github.com/seiji/menukey/internal/config"
	"github.com/seiji/menukey/internal/plan"
)

func chromePlan() plan.AppPlan {
	return plan.AppPlan{
		App: config.App{Bundle: "com.google.Chrome", Name: "Google Chrome"},
		Changes: []plan.Change{
			{Menu: "New Tab", Type: plan.Add, After: "ctrl+t", Encoded: "^t"},
			{Menu: "Close Tab", Type: plan.Unchanged, Before: "ctrl+w", After: "ctrl+w", Encoded: "^w"},
			{Menu: "Reopen Closed Tab", Type: plan.Update, Before: "cmd+shift+t", After: "ctrl+shift+t", Encoded: "^$t"},
		},
	}
}

func TestPrintPlans(t *testing.T) {
	var b strings.Builder
	pending := printPlans(&b, []plan.AppPlan{chromePlan()}, false, false)

	want := `Google Chrome (com.google.Chrome)

+ New Tab: ctrl+t
~ Reopen Closed Tab: cmd+shift+t -> ctrl+shift+t
`
	if b.String() != want {
		t.Errorf("output =\n%s\nwant\n%s", b.String(), want)
	}
	if pending != 2 {
		t.Errorf("pending = %d, want 2", pending)
	}
}

func TestPrintPlansShowUnchanged(t *testing.T) {
	var b strings.Builder
	printPlans(&b, []plan.AppPlan{chromePlan()}, true, false)

	if !strings.Contains(b.String(), "= Close Tab: ctrl+w") {
		t.Errorf("output does not list the unchanged shortcut:\n%s", b.String())
	}
}

func TestPrintPlansUpToDate(t *testing.T) {
	p := plan.AppPlan{
		App:     config.App{Bundle: "com.apple.Safari"},
		Changes: []plan.Change{{Menu: "New Tab", Type: plan.Unchanged, Before: "ctrl+t", After: "ctrl+t"}},
	}

	var b strings.Builder
	pending := printPlans(&b, []plan.AppPlan{p}, false, false)

	want := `com.apple.Safari

  (up to date)
`
	if b.String() != want {
		t.Errorf("output =\n%s\nwant\n%s", b.String(), want)
	}
	if pending != 0 {
		t.Errorf("pending = %d, want 0", pending)
	}
}

func TestPrintPlansSeparatesApps(t *testing.T) {
	second := plan.AppPlan{
		App:     config.App{Bundle: "com.apple.Safari"},
		Changes: []plan.Change{{Menu: "New Tab", Type: plan.Add, After: "ctrl+t"}},
	}

	var b strings.Builder
	printPlans(&b, []plan.AppPlan{chromePlan(), second}, false, false)

	if !strings.Contains(b.String(), "ctrl+shift+t\n\ncom.apple.Safari") {
		t.Errorf("apps are not separated by a blank line:\n%s", b.String())
	}
}

func TestPrintPlansShowUnmanaged(t *testing.T) {
	p := plan.AppPlan{
		App:     config.App{Bundle: "com.google.Chrome", Name: "Google Chrome"},
		Changes: []plan.Change{{Menu: "New Tab", Type: plan.Unchanged, Before: "ctrl+t", After: "ctrl+t"}},
		Unmanaged: []plan.UnmanagedShortcut{
			{Menu: "Reload This Page", Key: "ctrl+r"},
		},
	}

	var b strings.Builder
	printPlans(&b, []plan.AppPlan{p}, false, true)

	want := "! Reload This Page: ctrl+r (unmanaged)"
	if !strings.Contains(b.String(), want) {
		t.Errorf("output does not list the unmanaged shortcut:\n%s\nwant it to contain %q", b.String(), want)
	}
}

func TestPrintPlansHideUnmanagedByDefault(t *testing.T) {
	p := plan.AppPlan{
		App:     config.App{Bundle: "com.google.Chrome"},
		Changes: []plan.Change{{Menu: "New Tab", Type: plan.Unchanged, Before: "ctrl+t", After: "ctrl+t"}},
		Unmanaged: []plan.UnmanagedShortcut{
			{Menu: "Reload This Page", Key: "ctrl+r"},
		},
	}

	var b strings.Builder
	printPlans(&b, []plan.AppPlan{p}, false, false)

	if strings.Contains(b.String(), "unmanaged") {
		t.Errorf("output should not list unmanaged shortcuts without --unmanaged:\n%s", b.String())
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "1 change"},
		{2, "2 changes"},
		{0, "0 changes"},
	}
	for _, tt := range tests {
		if got := pluralize(tt.n, "change"); got != tt.want {
			t.Errorf("pluralize(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
