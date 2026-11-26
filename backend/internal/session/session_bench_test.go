package session

import (
	"testing"
	"time"
)

func BenchmarkSessionLifecycle(b *testing.B) {
	b.Run("create-destroy", func(b *testing.B) {
		s := NewStore(time.Hour)
		defer s.Close()
		b.ReportAllocs()
		for range b.N {
			v := s.Create(Principal{Role: "ADMIN", ID: 1})
			if v == nil {
				b.Fatal("create")
			}
			s.Destroy(v.ID)
		}
	})
	b.Run("get", func(b *testing.B) {
		s := NewStore(time.Hour)
		defer s.Close()
		v := s.Create(Principal{})
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if s.Get(v.ID) != v {
				b.Fatal("get")
			}
		}
	})
}
