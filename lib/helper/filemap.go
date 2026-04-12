package main

import (
	"strings"
)

func filemapAdd(origin, dest string) {
	fileMapLock.Lock()
	_, ok := fileSubstitutionMap[origin]
	if ok {
		delete(fileSubstitutionMap, origin)
	}
	fileSubstitutionMap[origin] = dest
	fileMapLock.Unlock()
}

func filemapDel(origin string) {
	fileMapLock.Lock()
	_, ok := fileSubstitutionMap[origin]
	if ok {
		delete(fileSubstitutionMap, origin)
	}
}

func filemapReplacer() *strings.Replacer {
	pairs := []string{}
	fileMapLock.RLock()
	for k, v := range fileSubstitutionMap {
		pairs = append(pairs, k, v)
	}
	fileMapLock.RUnlock()
	return strings.NewReplacer(pairs...)
}