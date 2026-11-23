package opscli

import (
    "bytes"
    "context"
    "database/sql"
    "strings"
    "testing"

    "suining-care/internal/securestore"
)

func TestRunBackfillHelpAndBounds(t *testing.T) {
    spec := BackfillSpec[struct{}]{
        Name: "test-backfill", ApplyUsage: "apply changes",
        LimitUsage: "maximum rows per invocation, 1..1000",
        Subject: "test backfill", RequireKeyOnScan: true,
        Run: func(_ context.Context, _ *sql.DB, _ *securestore.Box, _ int, _ bool) (struct{}, error) {
            t.Fatal("must reject before connecting to database")
            return struct{}{}, nil
        },
    }
    var out bytes.Buffer
    if err := RunBackfill(spec, []string{"--help"}, &out); err != nil { t.Fatal(err) }
    if !strings.Contains(out.String(), "1..1000") { t.Fatal("missing limit flag") }
    for _, args := range [][]string{{"--limit","0"}, {"--limit","1001"}, {"--limit","abc"}, {"--unknown"}, {"extra"}} {
        out.Reset()
        if err := RunBackfill(spec,args,&out); err == nil || out.Len() != 0 {
            t.Fatalf("args %v: err=%v out=%q", args,err,out.String())
        }
    }
}
