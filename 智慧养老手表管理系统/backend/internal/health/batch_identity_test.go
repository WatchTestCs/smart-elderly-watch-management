package health

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// Identity implementation at 738d84a, retained only for cost comparison.
func identityBeforeBatch(o Owner, s Sample) string {
	var stack [512]byte
	b := append(stack[:0], '[')
	b = strconv.AppendInt(b, o.OldID, 10)
	b = append(b, ',')
	if o.DeviceID == nil {
		b = append(b, "null"...)
	} else {
		b = strconv.AppendInt(b, *o.DeviceID, 10)
	}
	b = append(b, ',')
	b = strconv.AppendInt(b, o.Epoch, 10)
	b = appendJSONStringIdentity(b, s.Source)
	b = appendJSONStringIdentity(b, s.SourceDevice)
	b = appendJSONStringIdentity(b, s.RecordID)
	b = appendJSONStringIdentity(b, s.Metric)
	b = append(b, ']')
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:])
}

func TestBatchIdentityMatchesJSONAcrossOwnersAndSources(t *testing.T) {
	rng := rand.New(rand.NewSource(73884))
	device := int64(9223372036854775807)
	edges := []string{"", "watch", "引号\"\\\n", "设备🙂", "<>&\u2028\u2029", string([]byte{0xff, 0xfe}), strings.Repeat("长", 5000)}
	input := make([]preparedSample, 200)
	for round := range 30 {
		owner := Owner{OldID: rng.Int63(), Epoch: int64(round)}
		if round%2 == 0 {
			owner.DeviceID = &device
		}
		for i := range input {
			input[i] = preparedSample{revision: 999, sample: Sample{
				Source: edges[(i/3+round)%len(edges)], SourceDevice: edges[(i/5+round)%len(edges)],
				RecordID: fmt.Sprintf("%s-%d", edges[i%len(edges)], i), Metric: edges[(i+round)%len(edges)],
			}}
		}
		assignBatchIdentities(owner, input)
		for i := range input {
			if got, want := input[i].id, referenceIdentity(owner, input[i].sample); got != want || input[i].revision != 0 {
				t.Fatalf("round %d item %d: got=%s want=%s revision=%d", round, i, got, want, input[i].revision)
			}
		}
		device--
	}
}

func BenchmarkBatchIdentity(b *testing.B) {
	for _, size := range []int{1, 2, 8, 200} {
		for _, source := range []string{"ascii", "unicode", "mixed"} {
			b.Run(fmt.Sprintf("%d/%s", size, source), func(b *testing.B) {
				owner, sample := sampleBenchmarkInput()
				if source == "unicode" {
					sample.SourceDevice = "华为手表-设备🙂"
				}
				input := make([]preparedSample, size)
				for i := range input {
					input[i].sample = sample
					input[i].sample.RecordID = fmt.Sprintf("run-738d84a-%d", i)
					if source == "mixed" {
						input[i].sample.Source = []string{"watch", "healthkit"}[i%2]
					}
				}
				b.Run("individual", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						for i := range input {
							input[i].id = identityDigest(owner, input[i].sample)
							input[i].revision = 0
						}
					}
				})
				b.Run("baseline", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						for i := range input {
							input[i].id = identityBeforeBatch(owner, input[i].sample)
							input[i].revision = 0
						}
					}
				})
				b.Run("cached", func(b *testing.B) {
					b.ReportAllocs()
					for range b.N {
						assignBatchIdentities(owner, input)
					}
				})
			})
		}
	}
}
