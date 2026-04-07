package modbus

import "fmt"

func bytesFromWords(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, v := range words {
		out[i*2] = byte(v >> 8)
		out[i*2+1] = byte(v)
	}
	return out
}

func wordsFromBytes(results []byte) []uint16 {
	values := make([]uint16, len(results)/2)
	for i := 0; i+1 < len(results); i += 2 {
		values[i/2] = uint16(results[i])<<8 | uint16(results[i+1])
	}
	return values
}

func wordsFromBytesStrict(results []byte) ([]uint16, error) {
	if len(results)%2 != 0 {
		return nil, fmt.Errorf("odd byte count %d", len(results))
	}
	return wordsFromBytes(results), nil
}

func packedCoilsFromBools(values []bool) []byte {
	byteCount := (len(values) + 7) / 8
	out := make([]byte, byteCount)
	for i, v := range values {
		if v {
			out[i/8] |= 1 << uint(i%8)
		}
	}
	return out
}

func boolsFromPackedCoils(results []byte, quantity uint16) []bool {
	states := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < uint16(len(results)) {
			states[i] = (results[byteIndex] & (1 << bitIndex)) != 0
		}
	}
	return states
}
