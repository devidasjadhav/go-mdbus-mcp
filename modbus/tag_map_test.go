package modbus

import "testing"

func TestNewTagMapValidation(t *testing.T) {
	_, err := NewTagMap([]TagDef{{Name: "temp", Kind: TagKindHolding, Address: 0, Quantity: 1, Access: TagAccessRead}})
	if err != nil {
		t.Fatalf("expected valid tag map, got %v", err)
	}

	_, err = NewTagMap([]TagDef{{Name: "", Kind: TagKindHolding, Address: 0, Quantity: 1, Access: TagAccessRead}})
	if err == nil {
		t.Fatalf("expected empty name to fail")
	}

	_, err = NewTagMap([]TagDef{{Name: "bad", Kind: "weird", Address: 0, Quantity: 1, Access: TagAccessRead}})
	if err == nil {
		t.Fatalf("expected invalid kind to fail")
	}
}
