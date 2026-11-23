package main

import (
    "bytes"
    "strings"
    "testing"
)

func TestDispatchHelpAndUnknownTarget(t *testing.T) {
    var out bytes.Buffer
    if err := dispatch([]string{}, &out); err == nil { t.Fatal("missing usage error") }
    if err := dispatch([]string{"protect","unknown"}, &out); err == nil { t.Fatal("unknown target accepted") }
    for _, target := range []string{"family-addresses","family-phones","guardian-contacts","resident-birthdates","resident-profiles","social-phones"} {
        out.Reset()
        if err := dispatch([]string{"protect",target,"--help"}, &out); err != nil { t.Fatalf("%s: %v",target,err) }
        if !strings.Contains(out.String(),"1..1000") { t.Fatalf("%s: missing bounds",target) }
    }
}
