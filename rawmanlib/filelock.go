package rawmanlib

import (
	"sync"
)

var lockedMutex sync.Mutex
var lockedList map[string]bool = make(map[string]bool)

func lockFile(filename string) bool {
	lockedMutex.Lock()
	defer lockedMutex.Unlock()

	if lockedList[filename] {
		return false
	}

	lockedList[filename] = true
	return true
}

func unlockFile(filename string) {
	lockedMutex.Lock()
	defer lockedMutex.Unlock()
	delete(lockedList, filename)
}
