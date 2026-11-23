// Command protect-resident-profiles performs an explicit, bounded legacy data backfill.
package main

import (
    "fmt"
    "io"
    "os"

    "suining-care/internal/residentprofile"
    "suining-care/internal/opscli"
)

func main() {
    if err := run(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(args []string, output io.Writer) error {
    return opscli.RunBackfill(opscli.BackfillSpec[residentprofile.BackfillReport]{
        Name: "protect-resident-profiles",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum profiles per invocation, 1..1000",
        Subject: "resident backfill",
        RequireKeyOnScan: true,
        Run: residentprofile.Backfill,
    }, args, output)
}
