package common

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"
)

func Keccak256Bytes(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		_, _ = d.Write(b)
	}
	return d.Sum(nil)
}

func Keccak256String(content string) string {
	bs := Keccak256Bytes([]byte(content))
	return hex.EncodeToString(bs)
}
