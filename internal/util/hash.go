package util

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"

	"golang.org/x/exp/slices"
)

func Hash(values ...any) (string, error) {
	hash := sha256.New()
	for _, v := range values {
		if err := gob.NewEncoder(hash).Encode(v); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func StableIter[V any](m map[string]V) []string {
	var idx int
	keys := make([]string, len(m))
	for k := range m {
		keys[idx] = k
		idx++
	}
	slices.Sort(keys)
	return keys
}
