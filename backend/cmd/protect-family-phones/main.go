// Command protect-family-phones performs bounded, explicit legacy backfill.
package main

import (
	"fmt"
	"io"
	"os"

	"suining-care/internal/opscli"
	"suining-care/internal/userphone"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	return opscli.RunBackfill(opscli.BackfillSpec[userphone.FamilyBackfillReport]{
		Name:             "protect-family-phones",
		ApplyUsage:       "apply with account writers stopped and a tested backup",
		LimitUsage:       "maximum rows per invocation, 1..1000",
		Subject:          "family backfill",
		RequireKeyOnScan: false,
		Run:              userphone.BackfillFamily,
	}, args, output)
}
