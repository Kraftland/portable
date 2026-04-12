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
func filemapAddMap(pairs map[string]string) {
	fileMapLock.Lock()
	for k, v := range pairs {
		fileSubstitutionMap[k] = v
	}
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

func cmdlineReplacer(origin []string) []string {
	if len(fileSubstitutionMap) == 0 {
		return origin
	}
	replacer := filemapReplacer()
	var result = make([]string, 0, len(origin))
	for _, val := range origin {
		result = append(result, replacer.Replace(val))
	}
	return result
}