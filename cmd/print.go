package cmd

import (
	"fmt"
	"io"

	"github.com/seiji/menukey/internal/plan"
)

// printPlans writes the planned changes in diff form and returns how many
// changes are pending across every app.
//
//	Google Chrome (com.google.Chrome)
//
//	+ New Tab: ctrl+t
//	~ Reopen Closed Tab: cmd+shift+t -> ctrl+shift+t
//	! Reload This Page: ctrl+r (unmanaged)
func printPlans(w io.Writer, plans []plan.AppPlan, showUnchanged bool, showUnmanaged bool) int {
	pending := 0
	for i, p := range plans {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s\n\n", p.App.Label())

		printed := 0
		for _, c := range p.Changes {
			if c.Type == plan.Unchanged {
				if !showUnchanged {
					continue
				}
			} else {
				pending++
			}

			if c.Type == plan.Update {
				fmt.Fprintf(w, "%s %s: %s -> %s\n", c.Type.Symbol(), c.Menu, c.Before, c.After)
			} else {
				fmt.Fprintf(w, "%s %s: %s\n", c.Type.Symbol(), c.Menu, c.After)
			}
			printed++
		}

		if showUnmanaged {
			for _, u := range p.Unmanaged {
				fmt.Fprintf(w, "! %s: %s (unmanaged)\n", u.Menu, u.Key)
				printed++
			}
		}

		if printed == 0 {
			fmt.Fprintln(w, "  (up to date)")
		}
	}
	return pending
}
