package sftp

import (
	"sync"
)

type pagelist struct {
	pages [][]byte
}

type allocator struct {
	pool sync.Pool
	// map key is the request order
	used map[uint32]*pagelist
	sync.Mutex
}

func newAllocator() *allocator {
	return &allocator{
		pool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, maxMsgLength)
				return b
			},
		},
		used: make(map[uint32]*pagelist),
	}
}

// GetPage returns a previously allocated and unused []byte or create a new one.
func (a *allocator) GetPage(requestOrderID uint32) []byte {
	a.Lock()
	defer a.Unlock()

	result := a.pool.Get().([]byte)

	// put result in used page
	used := a.used[requestOrderID]
	if used == nil {
		used = new(pagelist)
		a.used[requestOrderID] = used
	}

	used.pages = append(used.pages, result)

	return result
}

// ReleasePages mark unused all pages in use for the given requestID
func (a *allocator) ReleasePages(requestOrderID uint32) {
	a.Lock()
	defer a.Unlock()

	if used := a.used[requestOrderID]; used != nil {
		for i := range used.pages {
			a.pool.Put(used.pages[i])
			used.pages[i] = nil // we want to clear out the internal pointer here, so it is not left hanging around.
		}
		delete(a.used, requestOrderID)
	}
}

func (a *allocator) Free() {
	a.Lock()
	defer a.Unlock()

	a.used = make(map[uint32]*pagelist)
}

func (a *allocator) countUsedPages() int {
	a.Lock()
	defer a.Unlock()

	num := 0
	for _, p := range a.used {
		if p != nil {
			num += len(p.pages)
		}
	}
	return num
}

func (a *allocator) isRequestOrderIDUsed(requestOrderID uint32) bool {
	a.Lock()
	defer a.Unlock()

	if _, ok := a.used[requestOrderID]; ok {
		return true
	}
	return false
}
