package sftp

import (
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllocator(t *testing.T) {
	allocator := newAllocator()
	page := allocator.GetPage(4, 1)
	page[1] = uint8(1)
	assert.Equal(t, 4, len(page))
	assert.Equal(t, 1, len(allocator.used[1][4].pages))
	page = allocator.GetPage(4, 1)
	page[0] = uint8(2)
	allocator.GetPage(2, 1)
	assert.Equal(t, 2, len(allocator.used[1][4].pages))
	assert.Equal(t, 1, len(allocator.used[1][2].pages))
	allocator.ReleasePages(1)
	assert.NotContains(t, allocator.used, 1)
	page = allocator.GetPage(4, 2)
	assert.Equal(t, uint8(2), page[0])
	page = allocator.GetPage(4, 2)
	assert.Equal(t, uint8(1), page[1])
	assert.Contains(t, allocator.available, 2)
	assert.Contains(t, allocator.used, uint32(2))
	assert.Equal(t, 1, len(allocator.used[2]))
	p := allocator.used[2][4]
	assert.Equal(t, 2, len(p.pages))
	allocator.ReleasePages(3)
	assert.Equal(t, 1, len(allocator.used[2]))
	allocator.ReleasePages(2)
	assert.Equal(t, 0, len(allocator.used))
	assert.Equal(t, 2, len(allocator.available))
	page = allocator.GetPage(4, 3)
	assert.Equal(t, uint8(1), page[1])
	page = allocator.GetPage(2, 4)
	allocator.ReleasePages(3)
	assert.Contains(t, allocator.used, uint32(4))
	assert.Equal(t, 1, len(allocator.used))
	assert.Contains(t, allocator.available, 4)
	assert.Equal(t, 2, len(allocator.available))
	assert.Equal(t, 0, len(allocator.available[2].pages))
	p = allocator.available[4]
	assert.Equal(t, 2, len(p.pages))
	allocator.Free()
	assert.Equal(t, 0, len(allocator.used))
	assert.Equal(t, 0, len(allocator.available))
}

func BenchmarkAllocatorSerial(b *testing.B) {
	allocator := newAllocator()
	for i := 0; i < b.N; i++ {
		benchAllocator(allocator, uint32(i))
	}
}

func BenchmarkAllocatorParallel(b *testing.B) {
	var counter uint32
	allocator := newAllocator()
	for i := 1; i <= 8; i *= 2 {
		b.Run(strconv.Itoa(i), func(b *testing.B) {
			b.SetParallelism(i)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					benchAllocator(allocator, atomic.AddUint32(&counter, 1))
				}
			})
		})
	}
}

func benchAllocator(allocator *allocator, requestOrderID uint32) {
	// simulates the page requested in recvPacket
	allocator.GetPage(maxMsgLength, requestOrderID)
	// simulates the page requested in fileget for downloads
	allocator.GetPage(int(maxTxPacket)+9+4, requestOrderID)
	// release the allocated pages
	allocator.ReleasePages(requestOrderID)
}

// useful for debug
func printAllocatorContents(allocator *allocator) {
	for o, u := range allocator.used {
		for k, v := range u {
			debug("used order id: %v, key: %v, values: %+v", o, k, v)
		}
	}
	for k, v := range allocator.available {
		debug("available, key: %v, values: %+v", k, v)
	}
}
