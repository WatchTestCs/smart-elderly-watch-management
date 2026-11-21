//go:build cgo && !purego

#include <algorithm>
#include <cstdint>

extern "C" void suining_sort_sleep(uint64_t *keys, uint32_t count) {
    std::sort(keys, keys + count);
}
