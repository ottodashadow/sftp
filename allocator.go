package sftp

import (
	"sync"
)

type pagelist struct {
	pages [][]byte
}

type allocator struct {
	// map key is the size of containted list of []byte
	available map[int]*pagelist
	// map key is the request order, the value is the same as available
	used map[uint32]map[int]*pagelist
	sync.Mutex
}

func newAllocator() *allocator {
	return &allocator{
		available: make(map[int]*pagelist),
		used:      make(map[uint32]map[int]*pagelist),
	}
}

// GetPage returns a previously allocated and unused []byte or create a new one.
func (a *allocator) GetPage(size int, requestOrderID uint32) []byte {
	a.Lock()
	defer a.Unlock()

	var result []byte
	if avail := a.available[size]; avail != nil {
		// get the available page and remove it from the available ones
		if len(avail.pages) > 0 {
			truncLength := len(avail.pages) - 1
			result = avail.pages[truncLength]

			avail.pages[truncLength] = nil          // clear out the internal pointer
			avail.pages = avail.pages[:truncLength] // truncate the slice
		}
	}

	// no preallocated slice found, doesn’t matter why, just alloc a new one
	if result == nil {
		result = make([]byte, size)
	}

	// put result in used page
	used := a.used[requestOrderID]
	if used == nil {
		used = make(map[int]*pagelist)
		a.used[requestOrderID] = used
	}

	page := used[size]
	if page == nil {
		page = new(pagelist)
		used[size] = page
	}
	page.pages = append(page.pages, result)

	return result
}

// ReleasePages mark unused all pages in use for the given requestID
func (a *allocator) ReleasePages(requestOrderID uint32) {
	a.Lock()
	defer a.Unlock()

	if used := a.used[requestOrderID]; used != nil {
		for size, p := range used {
			avail := a.available[size]
			if avail == nil {
				avail = new(pagelist)
				a.available[size] = avail
			}

			avail.pages = append(avail.pages, p.pages...)

			for i := range p.pages {
				p.pages[i] = nil // we want to clear out the internal pointer here, so it is not left hanging around.
			}
			// now we can just drop `p` on the floor. The garbage collector will clean it up.
			delete(used, size)
		}
		delete(a.used, requestOrderID)
	}
}

func (a *allocator) Free() {
	a.Lock()
	defer a.Unlock()

	a.available = make(map[int]*pagelist)
	a.used = make(map[uint32]map[int]*pagelist)
}

func (a *allocator) countUsedPages() int {
	a.Lock()
	defer a.Unlock()

	return len(a.used)
}

func (a *allocator) countAvailablePages() int {
	a.Lock()
	defer a.Unlock()

	return len(a.available)
}

func (a *allocator) hasUsedPageWithSize(size int) bool {
	a.Lock()
	defer a.Unlock()

	for _, v := range a.used {
		for k := range v {
			if k == size {
				return true
			}
		}
	}
	return false
}
