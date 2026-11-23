// Command suining-ctl groups offline data protection workflows.
package main

import (
    "fmt"
    "io"
    "os"

    "suining-care/internal/familyaddress"
    "suining-care/internal/guardiancontact"
    "suining-care/internal/residentprofile"
    "suining-care/internal/socialrepo"
    "suining-care/internal/userphone"
    "suining-care/internal/opscli"
)

func main() {
    if err := dispatch(os.Args[1:], os.Stdout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func dispatch(args []string, out io.Writer) error {
    if len(args) < 2 || args[0] != "protect" {
        return fmt.Errorf("usage: suining-ctl protect <family-addresses|family-phones|guardian-contacts|resident-birthdates|resident-profiles|social-phones> [--apply] [--limit N]")
    }
    target, rest := args[1], args[2:]
    switch target {
    case "family-addresses":
        return opscli.RunBackfill(opscli.BackfillSpec[familyaddress.BackfillReport]{
        Name: "suining-ctl protect family-addresses",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum addresses per invocation, 1..1000",
        Subject: "family address backfill",
        RequireKeyOnScan: true,
        Run: familyaddress.Backfill,
    }, rest, out)
    case "family-phones":
        return opscli.RunBackfill(opscli.BackfillSpec[userphone.FamilyBackfillReport]{
        Name: "suining-ctl protect family-phones",
        ApplyUsage: "apply with account writers stopped and a tested backup",
        LimitUsage: "maximum rows per invocation, 1..1000",
        Subject: "family backfill",
        RequireKeyOnScan: false,
        Run: userphone.BackfillFamily,
    }, rest, out)
    case "guardian-contacts":
        return opscli.RunBackfill(opscli.BackfillSpec[guardiancontact.BackfillReport]{
        Name: "suining-ctl protect guardian-contacts",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum contacts per invocation, 1..1000",
        Subject: "guardian contact backfill",
        RequireKeyOnScan: true,
        Run: guardiancontact.Backfill,
    }, rest, out)
    case "resident-birthdates":
        return opscli.RunBackfill(opscli.BackfillSpec[residentprofile.BackfillReport]{
        Name: "suining-ctl protect resident-birthdates",
        ApplyUsage: "apply with resident writers stopped and a tested backup",
        LimitUsage: "maximum birthdates per invocation, 1..1000",
        Subject: "resident birthdate backfill",
        RequireKeyOnScan: true,
        Run: residentprofile.BackfillBirthdates,
    }, rest, out)
    case "resident-profiles":
        return opscli.RunBackfill(opscli.BackfillSpec[residentprofile.BackfillReport]{
        Name: "suining-ctl protect resident-profiles",
        ApplyUsage: "apply with relevant writers stopped and a tested backup",
        LimitUsage: "maximum profiles per invocation, 1..1000",
        Subject: "resident backfill",
        RequireKeyOnScan: true,
        Run: residentprofile.Backfill,
    }, rest, out)
    case "social-phones":
        return opscli.RunBackfill(opscli.BackfillSpec[socialrepo.PhoneBackfillReport]{
        Name: "suining-ctl protect social-phones",
        ApplyUsage: "apply with social writers stopped and a tested backup",
        LimitUsage: "maximum rows per invocation, 1..1000",
        Subject: "social backfill",
        RequireKeyOnScan: true,
        Run: socialrepo.BackfillPhones,
    }, rest, out)
    default:
        return fmt.Errorf("unknown protect target %q", target)
    }
}
