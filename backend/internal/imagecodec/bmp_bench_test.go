package imagecodec

import (
	"encoding/binary"
	"fmt"
	"image"
	"testing"
)

func BenchmarkDecodeBMP(b *testing.B) {
	const width, height = 1920, 1080
	for _, channels := range []int{3, 4} {
		b.Run(fmt.Sprintf("BGR%d", channels*8), func(b *testing.B) {
			stride := (width*channels + 3) &^ 3
			data := make([]byte, 54+stride*height)
			copy(data, "BM")
			binary.LittleEndian.PutUint32(data[10:], 54)
			binary.LittleEndian.PutUint32(data[14:], 40)
			binary.LittleEndian.PutUint32(data[18:], width)
			binary.LittleEndian.PutUint32(data[22:], height)
			binary.LittleEndian.PutUint16(data[26:], 1)
			binary.LittleEndian.PutUint16(data[28:], uint16(8*channels))
			for i := 54; i < len(data); i++ {
				data[i] = byte(i * 17)
			}
			b.SetBytes(int64(stride * height))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, _, _, ok := decodeBMP(data); !ok {
					b.Fatal("decode failed")
				}
			}
		})
	}
}

func BenchmarkUnpackBMP(b *testing.B) {
	const width, height = 1920, 1080
	for _, channels := range []int{3, 4} {
		b.Run(fmt.Sprintf("BGR%d", channels*8), func(b *testing.B) {
			stride := (width*channels + 3) &^ 3
			src, dst := make([]byte, stride*height), make([]byte, width*height*4)
			for i := range src {
				src[i] = byte(i * 17)
			}
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if !unpackBMPNative(dst, src, width, height, stride, channels, true, 0xff0000, 0xff00, 0xff) {
					b.Skip("native path unavailable")
				}
			}
		})
	}
}

func BenchmarkDecodeBMPReusable(b *testing.B) {
	const width, height = 1280, 960
	data := newBMP(width, height, 24)
	for i := 54; i < len(data); i++ {
		data[i] = byte(i * 17)
	}
	for _, reuse := range []bool{false, true} {
		b.Run(fmt.Sprintf("reuse_%t", reuse), func(b *testing.B) {
			var dst *image.RGBA
			if reuse {
				dst = &image.RGBA{Pix: make([]byte, width*height*4)}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_, _, _, ok := decodeBMPInto(data, dst)
				if !ok {
					b.Fatal("decode")
				}
				if dst != nil {
					clear(dst.Pix)
				}
			}
		})
	}
}
