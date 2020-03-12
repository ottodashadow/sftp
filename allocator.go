package sftp

import (
	"sync"
)

type pagelist struct {
	pages [][]byte
}

type allocator struct {
	available *pagelist
	// map key is the request order
	used map[uint32]*pagelist
	sync.Mutex
}

func newAllocator() *allocator {
	return &allocator{
		available: new(pagelist),
		used:      make(map[uint32]*pagelist),
	}
}

// GetPage returns a previously allocated and unused []byte or create a new one.
func (a *allocator) GetPage(requestOrderID uint32) []byte {
	a.Lock()
	defer a.Unlock()

	var result []byte

	// get an available page and remove it from the available ones
	if len(a.available.pages) > 0 {
		truncLength := len(a.available.pages) - 1
		result = a.available.pages[truncLength]

		a.available.pages[truncLength] = nil                // clear out the internal pointer
		a.available.pages = a.available.pages[:truncLength] // truncate the slice
	}

	// no preallocated slice found, just allocate a new one
	if result == nil {
		result = make([]byte, maxMsgLength)
	}

	// put result in used pages
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
		a.available.pages = append(a.available.pages, used.pages...)
		for i := range used.pages {
			used.pages[i] = nil // we want to clear out the internal pointer here, so it is not left hanging around.
		}
		delete(a.used, requestOrderID)
	}
}

func (a *allocator) Free() {
	a.Lock()
	defer a.Unlock()

	a.available = new(pagelist)
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

func (a *allocator) countAvailablePages() int {
	a.Lock()
	defer a.Unlock()

	return len(a.available.pages)
}

func (a *allocator) isRequestOrderIDUsed(requestOrderID uint32) bool {
	a.Lock()
	defer a.Unlock()

	if _, ok := a.used[requestOrderID]; ok {
		return true
	}
	return false
}
