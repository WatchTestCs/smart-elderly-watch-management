// Command protect-guardian-contacts performs an explicit, bounded legacy data backfill.
package main

import (
    "fmt"
    "io"
    "os"

    "suining-care/internal/guardiancontact"
    "suining-care/internal/opscli"
)

func main() {
    if err := run(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(args []string, output io.Writer) error {
    return opscli.RunBackfill(opscli.BackfillSpec[guardiancontact.BackfillReport]{
        Name: "protect-guardian-contacts",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum contacts per invocation, 1..1000",
        Subject: "guardian contact backfill",
        RequireKeyOnScan: true,
        Run: guardiancontact.Backfill,
    }, args, output)
}
