package util

import (
	"crypto/md5"
	"encoding/base64"
	"math/rand"
	"time"
)

const _min = 15

// ID returns a base64 encoded id
func ID(salt string) string {
	if len(salt) < _min {
		salt = string(md5.New().Sum([]byte(salt)))
	}

	b64 := make([]byte, base64.StdEncoding.EncodedLen(_min))
	base64.StdEncoding.Encode(b64, []byte(salt[:_min]))
	rand.New(rand.NewSource(time.Now().UnixNano())).
		Shuffle(len(b64), func(i, j int) {
			x := b64[i]
			b64[i] = b64[j]
			b64[j] = x
		})

	return string(b64)
}
