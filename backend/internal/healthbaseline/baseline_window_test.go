package healthbaseline

import (
	"math/rand"
	"slices"
	"testing"
	"time"
)

func TestSparseUTCWindowAndInputPreservation(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 1, BindingVersion: 1, Metric: "heart_rate", At: now, Value: 90, Valid: true}
	var history []Observation
	for _, age := range []int{7, 28, 12, 10, 9, 20, 8} {
		for _, value := range []float64{64, 56} {
			s := current
			s.At = now.Truncate(24*time.Hour).AddDate(0, 0, -age).In(time.FixedZone("east", 8*3600))
			s.Value = value
			history = append(history, s)
		}
	}
	for _, age := range []int{0, 29, -1} {
		s := current
		s.At = now.AddDate(0, 0, -age)
		s.Value = 200
		history = append(history, s)
	}
	before := slices.Clone(history)
	got, err := Evaluate(current, history)
	want := Result{Ready: true, SampleDays: 7, TrustedDays: 7, Confidence: "moderate", Baseline: 60, Scale: 5, Score: 6, Direction: "above", Anomalous: true}
	if err != nil || got != want {
		t.Fatalf("got %+v, err=%v; want %+v", got, err, want)
	}
	if !slices.Equal(history, before) {
		t.Fatal("evaluation mutated input history")
	}
}

func TestDenseUnevenHistory(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 1, BindingVersion: 1, Metric: "heart_rate", At: now, Value: 90, Valid: true}
	var history []Observation
	for age := 1; age <= HistoryDays; age++ {
		n := 1
		if age == 1 {
			n = 1601
		} else if age == 2 {
			n = 25
		}
		for i := 0; i < n; i++ {
			s := current
			s.At = now.Truncate(24*time.Hour).AddDate(0, 0, -age)
			s.Value = 60
			history = append(history, s)
		}
	}
	before := slices.Clone(history)
	got, err := Evaluate(current, history)
	want := Result{Ready: true, SampleDays: 28, TrustedDays: 28, Confidence: "high", Baseline: 60, Scale: 5, Score: 6, Direction: "above", Anomalous: true}
	if err != nil || got != want || !slices.Equal(history, before) {
		t.Fatalf("got %+v, err=%v; want %+v, unchanged history", got, err, want)
	}
}

func TestGroupedHistoryAndFallback(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	current := Observation{OwnerID: 1, BindingVersion: 1, Metric: "heart_rate", At: now, Value: 90, Valid: true}
	var history []Observation
	for age := 1; age <= 8; age++ {
		for i := 0; i < 145; i++ {
			s := current
			s.At = now.Truncate(24*time.Hour).AddDate(0, 0, -age).Add(time.Duration(i) * time.Minute)
			s.Value = 60
			if age == 1 {
				s.Value = 100
			}
			history = append(history, s)
		}
	}
	reversed := slices.Clone(history)
	slices.Reverse(reversed)
	shuffled := slices.Clone(history)
	rng := rand.New(rand.NewSource(71))
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	late := append(slices.Clone(history), history[0])
	invalid := current
	invalid.Valid = false
	invalid.At = history[0].At
	late = append(late, invalid)
	want := Result{Ready: true, SampleDays: 8, TrustedDays: 7, RejectedDays: 1, Confidence: "guarded", Baseline: 60, Scale: 5, Score: 6, Direction: "above", Anomalous: true}
	for _, h := range [][]Observation{history, reversed, shuffled, late} {
		before := slices.Clone(h)
		got, err := Evaluate(current, h)
		if err != nil || got != want || !slices.Equal(h, before) {
			t.Fatalf("got %+v err=%v, want %+v unchanged input", got, err, want)
		}
	}
}
