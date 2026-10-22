// Offline CPU-only arrival/GC stress generator. A SaveBatch-style preparation
// workload retains all 200 encoded payloads until the batch completes.
// Does not model database I/O, network latency, or production SLAs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"perfbench/health"
)

type job struct{ queued time.Time }

const bucketWidth = 10000      // ten microseconds in nanoseconds
const histogramBuckets = 20001 // 200ms plus overflow

type histogram [histogramBuckets]uint64

func (h *histogram) observe(d time.Duration) {
	n := int(d.Nanoseconds() / bucketWidth)
	if n < 0 {
		n = 0
	}
	if n >= len(h) {
		n = len(h) - 1
	}
	h[n]++
}
func (h *histogram) quantile(q float64) float64 {
	total := uint64(0)
	for _, n := range h {
		total += n
	}
	if total == 0 {
		return 0
	}
	want := uint64(float64(total)*q + 0.999999)
	seen := uint64(0)
	for i, n := range h {
		seen += n
		if seen >= want {
			return float64((i+1)*bucketWidth) / 1000
		}
	}
	return float64(len(h)) * float64(bucketWidth) / 1000
}

type stats struct {
	Queue   histogram
	Service histogram
	Done    uint64
	Bytes   uint64
	Fail    uint64
}

type result struct {
	Mode                    string  `json:"mode"`
	GCPercent               int     `json:"gc_percent"`
	MemoryLimitEnv          string  `json:"gomemlimit_env"`
	Workers                 int     `json:"workers"`
	BatchSize               int     `json:"batch_size"`
	OfferedBatchesPerSecond int     `json:"offered_batches_per_second"`
	DurationSeconds         float64 `json:"duration_seconds"`
	CompletedBatches        uint64  `json:"completed_batches"`
	Accepted                uint64  `json:"accepted"`
	Dropped                 uint64  `json:"dropped"`
	MissedPacing            uint64  `json:"missed_pacing"`
	PacingLossPct           float64 `json:"pacing_loss_pct"`
	PacingValid             bool    `json:"pacing_valid"`
	SamplesPerSecond        float64 `json:"completed_samples_per_second"`
	P50Us                   float64 `json:"p50_us"`
	P95Us                   float64 `json:"p95_us"`
	P99Us                   float64 `json:"p99_us"`
	ServiceP99Us            float64 `json:"service_p99_us"`
	AllocatedBytes          uint64  `json:"allocated_bytes"`
	GCCount                 uint32  `json:"gc_count"`
	GCPauseMs               float64 `json:"gc_pause_ms"`
	HeapPeakBytes           uint64  `json:"heap_peak_bytes"`
	HeapFinalBytes          uint64  `json:"heap_final_bytes"`
	VmHWMB                  uint64  `json:"vm_hwm_bytes"`
	Failed                  uint64  `json:"failed"`
}

func rssHWM() uint64 {
	b, _ := os.ReadFile("/proc/self/status")
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmHWM:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				value, _ := strconv.ParseUint(fields[1], 10, 64)
				return value * 1024
			}
		}
	}
	return 0
}

func loadSamples() []health.Sample {
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	samples := make([]health.Sample, 200)
	for i := range samples {
		s := health.Sample{RecordID: fmt.Sprintf("sample-%04d", i), Source: "watch", SourceDevice: "WATCH-123456", Metric: "heart_rate", Unit: "count/min", Value: health.Number(72), Start: now, End: now, Version: 1}
		switch {
		case i%20 == 0:
			s.Metadata = map[string]string{"note": "care"}
		case i%10 == 0:
			s.SourceDevice = "北区老人手表"
		case i%7 == 0:
			s.SourceDevice = strings.Repeat("Z", 256)
		}
		samples[i] = s
	}
	return samples
}

func prepareBatch(samples []health.Sample) (uint64, error) {

	type prepared struct {
		hash string
		body []byte
	}
	outputs := make([]prepared, 0, len(samples))
	var length uint64
	for _, s := range samples {
		n, err := s.Normalize()
		if err != nil {
			return 0, err
		}
		body, err := health.MarshalBenchmark(n)
		if err != nil {
			return 0, err
		}
		id := health.IdentityBenchmark(n)
		outputs = append(outputs, prepared{hash: id, body: body})
		length += uint64(len(body) + len(id))
	}
	runtime.KeepAlive(outputs)
	return length, nil
}

func main() {
	mode := flag.String("mode", "baseline", "baseline")
	workers := flag.Int("workers", 4, "number of CPU workers")
	duration := flag.Duration("duration", 12*time.Second, "time to admit work")
	rate := flag.Int("rate", 0, "offered batches per second (0=unbounded)")
	gcPercent := flag.Int("gogc", 100, "runtime GC percent")
	flag.Parse()
	if *mode != "baseline" {
		panic("invalid mode")
	}
	runtime.GOMAXPROCS(*workers)
	debug.SetGCPercent(*gcPercent)
	samples := loadSamples()
	jobs := make(chan job, 128)
	collected := make([]stats, *workers)
	var startMS, endMS runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMS)
	var heapPeak atomic.Uint64
	quit := make(chan struct{})
	var monitor sync.WaitGroup
	monitor.Add(1)
	go func() {
		defer monitor.Done()
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-quit:
				return
			case <-t.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				for {
					old := heapPeak.Load()
					if m.HeapAlloc <= old || heapPeak.CompareAndSwap(old, m.HeapAlloc) {
						break
					}
				}
			}
		}
	}()
	var wg sync.WaitGroup
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			st := &collected[w]
			for j := range jobs {
				start := time.Now()
				bytes, err := prepareBatch(samples)
				end := time.Now()
				st.Queue.observe(end.Sub(j.queued))
				st.Service.observe(end.Sub(start))
				st.Done++
				st.Bytes += bytes
				if err != nil {
					st.Fail++
				}
			}
		}(w)
	}
	start := time.Now()
	var accepted, dropped, missedPacing uint64
	if *rate == 0 {
		// Closed loop: workers generate their own work. No spinning producer,
		// channel contention, or fake queue-induced p99 at peak throughput.
		close(jobs)
		wg.Wait()
		var direct sync.WaitGroup
		deadline := time.Now().Add(*duration)
		for w := 0; w < *workers; w++ {
			direct.Add(1)
			go func(w int) {
				defer direct.Done()
				st := &collected[w]
				for time.Now().Before(deadline) {
					s := time.Now()
					bytes, err := prepareBatch(samples)
					end := time.Now()
					st.Queue.observe(end.Sub(s))
					st.Service.observe(end.Sub(s))
					st.Done++
					st.Bytes += bytes
					if err != nil {
						st.Fail++
					}
				}
			}(w)
		}
		direct.Wait()
		for i := range collected {
			accepted += collected[i].Done
		}
	} else {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		deadline := start.Add(*duration)
		for time.Now().Before(deadline) {
			<-ticker.C
			// Delayed ticks are missed by the producer, not queued as a massive
			// synthetic burst. Count those arrivals separately from queue drops.
			want := uint64(time.Since(start).Nanoseconds()) * uint64(*rate) / uint64(time.Second)
			due := want - min(want, accepted+dropped+missedPacing)
			maxBurst := uint64(max(2, (*rate/1000)*3))
			if due > maxBurst {
				missedPacing += due - maxBurst
				due = maxBurst
			}
			for i := uint64(0); i < due; i++ {
				j := job{queued: time.Now()}
				select {
				case jobs <- j:
					accepted++
				default:
					dropped++
				}
			}
		}
		close(jobs)
		wg.Wait()
	}
	elapsed := time.Since(start).Seconds()

	close(quit)
	monitor.Wait()
	runtime.ReadMemStats(&endMS)
	var q, s histogram
	var done, fail uint64
	for i := range collected {
		w := &collected[i]
		done += w.Done
		fail += w.Fail
		for b, n := range w.Queue {
			q[b] += n
		}
		for b, n := range w.Service {
			s[b] += n
		}
	}
	offered := accepted + dropped + missedPacing
	pacingLoss := 0.0
	if offered > 0 {
		pacingLoss = 100 * float64(missedPacing) / float64(offered)
	}
	r := result{PacingLossPct: pacingLoss, PacingValid: *rate == 0 || (pacingLoss <= 2 && dropped == 0), Mode: *mode, GCPercent: *gcPercent, MemoryLimitEnv: os.Getenv("GOMEMLIMIT"), Workers: *workers, BatchSize: len(samples), OfferedBatchesPerSecond: *rate, DurationSeconds: elapsed, CompletedBatches: done, Accepted: accepted, Dropped: dropped, MissedPacing: missedPacing, SamplesPerSecond: float64(done*uint64(len(samples))) / elapsed, P50Us: q.quantile(.5), P95Us: q.quantile(.95), P99Us: q.quantile(.99), ServiceP99Us: s.quantile(.99), AllocatedBytes: endMS.TotalAlloc - startMS.TotalAlloc, GCCount: endMS.NumGC - startMS.NumGC, GCPauseMs: float64(endMS.PauseTotalNs-startMS.PauseTotalNs) / 1e6, HeapPeakBytes: heapPeak.Load(), HeapFinalBytes: endMS.HeapAlloc, VmHWMB: rssHWM(), Failed: fail}
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
}
