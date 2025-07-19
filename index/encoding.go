package index

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	ConcreteQueryPrefix byte = 0x00 // query in its original form
	AbstractQueryPrefix byte = 0x01 // query in its generalized form
	AuxDataPrefix       byte = 0x02 // auxiliary data
)

func encodeOriginalQuery(lemma string) []byte {
	lemmaBytes := []byte(lemma)
	key := make([]byte, 1+len(lemmaBytes))
	key[0] = ConcreteQueryPrefix
	copy(key[1:], lemmaBytes)
	return key
}

func encodeFrequency(freq uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, freq)
	return buf
}

func decodeFrequency(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}

func encodeTime(t time.Time) []byte {
	buf := make([]byte, 16) // 8 bytes for seconds + 8 bytes for nanoseconds

	// Convert to UTC to ensure consistency
	utc := t.UTC()

	// Store Unix timestamp (seconds since epoch)
	binary.BigEndian.PutUint64(buf[0:8], uint64(utc.Unix()))

	// Store nanoseconds component
	binary.BigEndian.PutUint64(buf[8:16], uint64(utc.Nanosecond()))

	return buf
}

func decodeTime(data []byte) (time.Time, error) {
	if len(data) != 16 {
		return time.Time{}, fmt.Errorf("invalid byte slice length: expected 16, got %d", len(data))
	}

	// Extract seconds
	seconds := int64(binary.BigEndian.Uint64(data[0:8]))

	// Extract nanoseconds
	nanoseconds := int64(binary.BigEndian.Uint64(data[8:16]))

	// Reconstruct time in UTC
	return time.Unix(seconds, nanoseconds).UTC(), nil
}
