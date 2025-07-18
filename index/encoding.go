package index

import "encoding/binary"

const (
	ConcreteQueryPrefix byte = 0x00 // query in its original form
	AbstractQueryPrefix byte = 0x01 // query in its generalized form
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
