// Command protect-family-addresses performs an explicit, bounded legacy data backfill.
package main

import (
    "fmt"
    "io"
    "os"

    "suining-care/internal/familyaddress"
    "suining-care/internal/opscli"
)

func main() {
    if err := run(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(args []string, output io.Writer) error {
    return opscli.RunBackfill(opscli.BackfillSpec[familyaddress.BackfillReport]{
        Name: "protect-family-addresses",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum addresses per invocation, 1..1000",
        Subject: "family address backfill",
        RequireKeyOnScan: true,
        Run: familyaddress.Backfill,
    }, args, output)
}
