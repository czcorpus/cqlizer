package stats

import (
	"crypto/sha1"
	"encoding/hex"
	"time"
)

func IdempotentID(created time.Time, query string) string {
	sum := sha1.New()
	_, err := sum.Write([]byte(created.String() + "#"))
	if err != nil {
		panic("problem generating hash")
	}
	_, err = sum.Write([]byte(query))
	if err != nil {
		panic("problem generating hash")
	}

	return hex.EncodeToString(sum.Sum(nil))
}
