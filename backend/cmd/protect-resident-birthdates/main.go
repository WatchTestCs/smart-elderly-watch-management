// Command protect-resident-birthdates explicitly migrates legacy dates of birth.
package main

import (
	"fmt"
	"io"
	"os"

	"suining-care/internal/opscli"
	"suining-care/internal/residentprofile"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	return opscli.RunBackfill(opscli.BackfillSpec[residentprofile.BackfillReport]{
		Name:             "protect-resident-birthdates",
		ApplyUsage:       "apply with resident writers stopped and a tested backup",
		LimitUsage:       "maximum birthdates per invocation, 1..1000",
		Subject:          "resident birthdate backfill",
		RequireKeyOnScan: true,
		Run:              residentprofile.BackfillBirthdates,
	}, args, output)
}
