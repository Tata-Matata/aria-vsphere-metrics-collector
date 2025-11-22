package util

import (
	"cmp"
	"slices"
	"strings"
)

// sorted keys from a map
func SortedKeysFromMap[T cmp.Ordered](amap map[T]T) []T {
	if amap == nil {
		return []T{}
	}
	keys := make([]T, 0, len(amap))
	for k := range amap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}

// merges key-value pairs from map into a single string
func JoinMapEntries(labels map[string]string, separator string) string {
	if separator == "" {
		separator = "|"
	}
	keys := SortedKeysFromMap(labels)
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, "|")
}
