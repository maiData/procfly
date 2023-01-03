package util

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
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
