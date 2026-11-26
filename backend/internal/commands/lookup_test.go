package commands

import (
	"strings"
	"testing"
)

// Source: DeviceType.java constructors and ShowUtils.java:823-841 in the reference
// bundle. Values below are getType() values, never Java enum identifiers.
const fourG = "C3 C3_PRO M1_A M1_B M1_PRO UE U1 UE_C GW03 X1 X1_B X1_C X1_S X1_A X1_PRO M3_CARE X3 X3D X5 X5_S X5_A X7 X8 X8_A X8_B X8_C HA07X HA07X_AI HA07X_433 HA07X_B"
const gprs = "S1 BY102 K1 BY102_LOC BY102_NO_HR C16 S3 S5 S7 BY007 BY007C SHIJIE_GPRS_BP T007 HA01 JIEMEIRUI_WIFI_BP"

func TestChannelLookupAllWireTypes(t *testing.T) {
	seen := map[string]bool{"S6T": true}
	for _, group := range []struct{ values, channel string }{{fourG, "4g"}, {gprs, "gprs"}} {
		for _, typ := range strings.Fields(group.values) {
			if seen[typ] {
				t.Fatalf("duplicate wire type %s", typ)
			}
			seen[typ] = true
			got, err := Channel(typ)
			if err != nil || got != group.channel {
				t.Fatalf("type=%s got=%s err=%v", typ, got, err)
			}
			for _, unsupported := range []string{strings.ToLower(typ), " " + typ, typ + " "} {
				if _, err := Channel(unsupported); err == nil {
					t.Fatalf("accepted %q", unsupported)
				}
			}
		}
	}
	if got, err := Channel("S6T"); err != nil || got != "s6t" {
		t.Fatal("S6T lookup changed")
	}
	if _, err := Channel(""); err == nil {
		t.Fatal("empty wire type accepted")
	}
}

func BenchmarkResolveCommand(b *testing.B) {
	for _, typ := range []string{"X1", "S1", "S6T"} {
		b.Run(typ, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, _, err := Resolve(typ, "location"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
