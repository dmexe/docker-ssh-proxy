package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
)

// StringDigest sorts and compute sha1 digest for given strings
func StringDigest(str ...string) string {
	sort.Strings(str)
	dig := sha1.New()
	for _, it := range str {
		dig.Write([]byte(it))
		dig.Write([]byte{0})
	}
	return hex.EncodeToString(dig.Sum(nil))
}
