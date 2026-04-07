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
	Name     string    `json:"name"`
	Kind     TagKind   `json:"kind"`
	Address  uint16    `json:"address"`
	Quantity uint16    `json:"quantity"`
	SlaveID  *uint8    `json:"slave_id,omitempty"`
	Access   TagAccess `json:"access"`
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
		if tag.Name == "" {
			return nil, fmt.Errorf("tag name must not be empty")
		}
		if _, exists := byName[tag.Name]; exists {
			return nil, fmt.Errorf("duplicate tag name %q", tag.Name)
		}
		if tag.Quantity == 0 {
			return nil, fmt.Errorf("tag %q quantity must be greater than 0", tag.Name)
		}
		if tag.Kind != TagKindHolding && tag.Kind != TagKindCoil {
			return nil, fmt.Errorf("tag %q has invalid kind %q", tag.Name, tag.Kind)
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
