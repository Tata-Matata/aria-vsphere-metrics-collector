package util

import (
	"cmp"
	"slices"
	"strings"

	"github.com/Tata-Matata/aria-vsphere-metrics-collector/logger"
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
// map to "errType=unathenticated|status=failure"
func JoinMapEntries(labels map[string]string) string {

	keys := SortedKeysFromMap(labels)
	if len(keys) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+KEY_VAL_SEPARATOR+labels[k])
	}
	return strings.Join(parts, MAP_ENTRY_SEPARATOR)
}

// merges key-value pairs from map into a single string
// "errType=unathenticated|status=failure" to map
func MapFromString(mapAsString string) map[string]string {
	labels := make(map[string]string)
	if mapAsString == "" {
		logger.Error("Joined labels string is empty")
		return labels
	}

	pairs := strings.Split(mapAsString, MAP_ENTRY_SEPARATOR)

	for _, keyVal := range pairs {
		key := strings.Split(keyVal, KEY_VAL_SEPARATOR)[0]
		value := strings.Split(keyVal, KEY_VAL_SEPARATOR)[1]
		labels[key] = value
	}
	return labels
}
