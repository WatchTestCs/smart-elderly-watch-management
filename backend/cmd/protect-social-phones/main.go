// Command protect-social-phones performs an explicit, bounded legacy data backfill.
package main

import (
    "fmt"
    "io"
    "os"

    "suining-care/internal/socialrepo"
    "suining-care/internal/opscli"
)

func main() {
    if err := run(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(args []string, output io.Writer) error {
    return opscli.RunBackfill(opscli.BackfillSpec[socialrepo.PhoneBackfillReport]{
        Name: "protect-social-phones",
        ApplyUsage: "apply with social writers stopped and a tested backup",
        LimitUsage: "maximum rows per invocation, 1..1000",
        Subject: "social backfill",
        RequireKeyOnScan: true,
        Run: socialrepo.BackfillPhones,
    }, args, output)
}
