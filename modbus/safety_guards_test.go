package modbus

import "testing"

func TestWordsFromBytesStrictRejectsOddLength(t *testing.T) {
	_, err := wordsFromBytesStrict([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatalf("expected odd-length byte slice to fail")
	}
}

func TestWordsFromBytesStrictAcceptsEvenLength(t *testing.T) {
	words, err := wordsFromBytesStrict([]byte{0x12, 0x34, 0xAB, 0xCD})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(words) != 2 || words[0] != 0x1234 || words[1] != 0xABCD {
		t.Fatalf("unexpected words: %v", words)
	}
}

func TestSimonvetterWriteMultipleCoilsRejectsShortPayload(t *testing.T) {
	d := &simonvetterDriver{}
	_, err := d.WriteMultipleCoils(0, 10, []byte{0x01})
	if err == nil {
		t.Fatalf("expected short payload to fail")
	}
}

func TestSimonvetterWriteMultipleCoilsRejectsZeroQuantity(t *testing.T) {
	d := &simonvetterDriver{}
	_, err := d.WriteMultipleCoils(0, 0, nil)
	if err == nil {
		t.Fatalf("expected zero quantity to fail")
	}
}
