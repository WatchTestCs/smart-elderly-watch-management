//go:build cgo && !purego

#include <stddef.h>
#include <stdint.h>
#include <tmmintrin.h>

struct channel {
    uint32_t mask;
    unsigned shift;
    unsigned width;
    uint32_t divisor;
};

static struct channel prepare_channel(uint32_t mask) {
    if (mask == 0) return (struct channel){0, 0, 0, 1};
    unsigned width = __builtin_popcount(mask);
    return (struct channel){mask, __builtin_ctz(mask), width, width < 8 ? (1u << width) - 1 : 1};
}

static uint8_t extract_channel(uint32_t pixel, struct channel channel) {
    uint32_t value = (pixel & channel.mask) >> channel.shift;
    if (channel.width >= 8) return (uint8_t)(value >> (channel.width - 8));
    return (uint8_t)(value * UINT32_C(255) / channel.divisor);
}

// Each vector load is wholly inside the input row. For BGR24 only four
// pixels are consumed, so up to five final pixels use the scalar tail.
__attribute__((target("ssse3")))
static void unpack_bgr24_ssse3(uint8_t *out, const uint8_t *src, int width) {
    const __m128i alpha = _mm_set1_epi32((int)UINT32_C(0xff000000));
    const __m128i shuffle = _mm_setr_epi8(2,1,0,-1,5,4,3,-1,8,7,6,-1,11,10,9,-1);
    int x = 0;
    for (; width - x >= 6; x += 4) {
        __m128i pixels = _mm_loadu_si128((const __m128i *)(src + x * 3));
        _mm_storeu_si128((__m128i *)(out + x * 4), _mm_or_si128(_mm_shuffle_epi8(pixels, shuffle), alpha));
    }
    for (; x < width; ++x) {
        out[4*x] = src[3*x+2];
        out[4*x+1] = src[3*x+1];
        out[4*x+2] = src[3*x];
        out[4*x+3] = 255;
    }
}

// The caller has checked the sizes, row stride, and non-overlapping buffers.
void suining_unpack_bmp(uint8_t *restrict out, const uint8_t *restrict src,
                        int width, int height, int stride, int channels, int top_down, uint32_t red, uint32_t green, uint32_t blue) {
    const int standard = red == 0xff0000 && green == 0xff00 && blue == 0xff;
    if (channels == 3 && __builtin_cpu_supports("ssse3")) {
        for (int y = 0; y < height; ++y) {
            const int row = top_down ? y : height - 1 - y;
            unpack_bgr24_ssse3(out + (size_t)y * width * 4, src + (size_t)row * stride, width);
        }
        return;
    }
    const struct channel r = prepare_channel(red), g = prepare_channel(green), b = prepare_channel(blue);
    for (int y = 0; y < height; ++y) {
        const int row = top_down ? y : height - 1 - y;
        const uint8_t *input = src + (size_t)row * stride;
        uint8_t *output = out + (size_t)y * width * 4;
        if (channels == 3) {
            for (int x = 0; x < width; ++x) {
                output[4*x] = input[3*x+2];
                output[4*x+1] = input[3*x+1];
                output[4*x+2] = input[3*x];
                output[4*x+3] = 255;
            }
        } else if (standard) {
            for (int x = 0; x < width; ++x) {
                output[4*x] = input[4*x+2];
                output[4*x+1] = input[4*x+1];
                output[4*x+2] = input[4*x];
                output[4*x+3] = 255;
            }
        } else {
            for (int x = 0; x < width; ++x) {
                const uint8_t *pixel = input + 4*x;
                uint32_t value = (uint32_t)pixel[0] | (uint32_t)pixel[1] << 8 |
                                 (uint32_t)pixel[2] << 16 | (uint32_t)pixel[3] << 24;
                output[4*x] = extract_channel(value, r);
                output[4*x+1] = extract_channel(value, g);
                output[4*x+2] = extract_channel(value, b);
                output[4*x+3] = 255;
            }
        }
    }
}
