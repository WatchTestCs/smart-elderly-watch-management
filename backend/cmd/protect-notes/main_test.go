package main

import (
    "bytes"
    "strings"
    "testing"
)

func TestProtectNotesHelpAndBounds(t *testing.T) {
    var out bytes.Buffer
    if err := run([]string{"--help"},&out); err != nil { t.Fatal(err) }
    if !strings.Contains(out.String(),"1..1000") { t.Fatal("limit help missing") }
    for _, args := range [][]string{{"--limit","0"},{"--limit","1001"},{"--limit","bad"},{"--unknown"},{"extra"}} {
        out.Reset()
        if err := run(args,&out); err == nil || out.Len() != 0 {
            t.Fatalf("%v: err=%v out=%q",args,err,out.String())
        }
    }
}
