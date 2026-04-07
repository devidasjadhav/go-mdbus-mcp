package modbus

import (
	"fmt"
	"sort"
	"strings"
)

type TagKind string

const (
	TagKindHolding TagKind = "holding_register"
	TagKindCoil    TagKind = "coil"
)

type TagAccess string

const (
	TagAccessRead      TagAccess = "read"
	TagAccessWrite     TagAccess = "write"
	TagAccessReadWrite TagAccess = "read_write"
)

// TagDef describes a semantic Modbus tag.
type TagDef struct {
	Name        string    `json:"name"`
	Kind        TagKind   `json:"kind"`
	Address     uint16    `json:"address"`
	Quantity    uint16    `json:"quantity"`
	SlaveID     *uint8    `json:"slave_id,omitempty"`
	Access      TagAccess `json:"access"`
	DataType    string    `json:"data_type,omitempty"`
	ByteOrder   string    `json:"byte_order,omitempty"`
	WordOrder   string    `json:"word_order,omitempty"`
	Scale       float64   `json:"scale,omitempty"`
	Offset      float64   `json:"offset,omitempty"`
	Description string    `json:"description,omitempty"`
	ScaleSet    bool      `json:"-"`
}

type TagMap struct {
	byName map[string]TagDef
	list   []TagDef
}

func NewTagMap(tags []TagDef) (*TagMap, error) {
	if len(tags) == 0 {
		return nil, nil
	}

	byName := make(map[string]TagDef, len(tags))
	for _, tag := range tags {
		tag.Name = strings.TrimSpace(tag.Name)
		tag.Description = strings.TrimSpace(tag.Description)
		tag.Kind = TagKind(strings.ToLower(strings.TrimSpace(string(tag.Kind))))
		tag.Access = TagAccess(strings.ToLower(strings.TrimSpace(string(tag.Access))))
		if tag.Name == "" {
			return nil, fmt.Errorf("tag name must not be empty")
		}
		if _, exists := byName[tag.Name]; exists {
			return nil, fmt.Errorf("duplicate tag name %q", tag.Name)
		}
		tag.DataType = normalizeDataType(tag.DataType)
		tag.ByteOrder = normalizeByteOrder(tag.ByteOrder)
		tag.WordOrder = normalizeWordOrder(tag.WordOrder)
		if tag.Kind == TagKindCoil && tag.DataType == "" {
			tag.DataType = "bool"
		}
		if tag.Kind == TagKindHolding && tag.DataType == "" {
			tag.DataType = "uint16"
		}
		if !tag.ScaleSet {
			tag.Scale = 1
		}

		expectedQty := expectedQuantity(tag.Kind, tag.DataType)
		if tag.Quantity == 0 {
			if expectedQty == 0 {
				return nil, fmt.Errorf("tag %q quantity must be provided", tag.Name)
			}
			tag.Quantity = expectedQty
		}
		if expectedQty > 0 && tag.Quantity != expectedQty {
			return nil, fmt.Errorf("tag %q quantity %d does not match data_type %s (expected %d)", tag.Name, tag.Quantity, tag.DataType, expectedQty)
		}

		if tag.Kind != TagKindHolding && tag.Kind != TagKindCoil {
			return nil, fmt.Errorf("tag %q has invalid kind %q", tag.Name, tag.Kind)
		}
		if err := validateDataType(tag); err != nil {
			return nil, err
		}
		if tag.Access == "" {
			tag.Access = TagAccessRead
		}
		if tag.Access != TagAccessRead && tag.Access != TagAccessWrite && tag.Access != TagAccessReadWrite {
			return nil, fmt.Errorf("tag %q has invalid access %q", tag.Name, tag.Access)
		}

		byName[tag.Name] = tag
	}

	list := make([]TagDef, 0, len(byName))
	for _, tag := range byName {
		list = append(list, tag)
	}
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return &TagMap{byName: byName, list: list}, nil
}

func (tm *TagMap) Get(name string) (TagDef, bool) {
	if tm == nil {
		return TagDef{}, false
	}
	tag, ok := tm.byName[strings.TrimSpace(name)]
	return tag, ok
}

func (tm *TagMap) List() []TagDef {
	if tm == nil {
		return nil
	}
	out := make([]TagDef, len(tm.list))
	copy(out, tm.list)
	return out
}

func (t TagDef) Readable() bool {
	return t.Access == TagAccessRead || t.Access == TagAccessReadWrite
}

func (t TagDef) Writable() bool {
	return t.Access == TagAccessWrite || t.Access == TagAccessReadWrite
}

func normalizeDataType(dataType string) string {
	v := strings.ToLower(strings.TrimSpace(dataType))
	if v == "" {
		return ""
	}
	return v
}

func normalizeByteOrder(byteOrder string) string {
	v := strings.ToLower(strings.TrimSpace(byteOrder))
	if v == "" {
		return "big"
	}
	if v != "big" && v != "little" {
		return ""
	}
	return v
}

func normalizeWordOrder(wordOrder string) string {
	v := strings.ToLower(strings.TrimSpace(wordOrder))
	if v == "" {
		return "msw"
	}
	if v != "msw" && v != "lsw" {
		return ""
	}
	return v
}

func expectedQuantity(kind TagKind, dataType string) uint16 {
	if kind == TagKindCoil {
		return 1
	}
	switch dataType {
	case "", "uint16", "int16":
		return 1
	case "uint32", "int32", "float32":
		return 2
	case "string":
		return 0
	default:
		return 0
	}
}

func validateDataType(tag TagDef) error {
	if tag.ByteOrder == "" {
		return fmt.Errorf("tag %q has invalid byte_order", tag.Name)
	}
	if tag.WordOrder == "" {
		return fmt.Errorf("tag %q has invalid word_order", tag.Name)
	}

	if tag.Kind == TagKindCoil {
		if tag.DataType == "" {
			tag.DataType = "bool"
		}
		if tag.DataType != "bool" {
			return fmt.Errorf("tag %q coil supports only data_type=bool", tag.Name)
		}
		return nil
	}

	if tag.DataType == "" {
		tag.DataType = "uint16"
	}
	switch tag.DataType {
	case "uint16", "int16", "uint32", "int32", "float32", "string":
		return nil
	default:
		return fmt.Errorf("tag %q has unsupported data_type %q", tag.Name, tag.DataType)
	}
}
