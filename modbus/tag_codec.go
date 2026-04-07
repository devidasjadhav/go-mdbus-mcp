package modbus

import (
	"fmt"
	"math"
	"strings"
)

func decodeHoldingTagValue(tag TagDef, regs []uint16) (any, error) {
	if len(regs) == 0 {
		return nil, fmt.Errorf("no registers to decode")
	}

	dataType := tag.DataType
	if dataType == "" {
		dataType = "uint16"
	}

	words := make([]uint16, len(regs))
	copy(words, regs)

	for i := range words {
		words[i] = applyByteOrder(words[i], tag.ByteOrder)
	}
	if len(words) == 2 && tag.WordOrder == "lsw" {
		words[0], words[1] = words[1], words[0]
	}

	switch dataType {
	case "uint16":
		v := float64(words[0])
		return applyScaleOffset(v, tag), nil
	case "int16":
		v := float64(int16(words[0]))
		return applyScaleOffset(v, tag), nil
	case "uint32":
		if len(words) < 2 {
			return nil, fmt.Errorf("uint32 requires 2 registers")
		}
		val := uint32(words[0])<<16 | uint32(words[1])
		return applyScaleOffset(float64(val), tag), nil
	case "int32":
		if len(words) < 2 {
			return nil, fmt.Errorf("int32 requires 2 registers")
		}
		u := uint32(words[0])<<16 | uint32(words[1])
		val := int32(u)
		return applyScaleOffset(float64(val), tag), nil
	case "float32":
		if len(words) < 2 {
			return nil, fmt.Errorf("float32 requires 2 registers")
		}
		u := uint32(words[0])<<16 | uint32(words[1])
		val := float64(math.Float32frombits(u))
		return applyScaleOffset(val, tag), nil
	case "string":
		bytes := make([]byte, 0, len(words)*2)
		for _, w := range words {
			bytes = append(bytes, byte(w>>8), byte(w&0xFF))
		}
		return strings.TrimRight(string(bytes), "\x00 "), nil
	default:
		return nil, fmt.Errorf("unsupported data_type %q", dataType)
	}
}

func applyByteOrder(word uint16, byteOrder string) uint16 {
	if byteOrder == "little" {
		return (word>>8)&0x00FF | (word<<8)&0xFF00
	}
	return word
}

func applyScaleOffset(v float64, tag TagDef) float64 {
	scale := tag.Scale
	if !tag.ScaleSet {
		scale = 1
	}
	return v*scale + tag.Offset
}

func removeScaleOffset(v float64, tag TagDef) (float64, error) {
	scale := tag.Scale
	if !tag.ScaleSet {
		scale = 1
	}
	if scale == 0 {
		return 0, fmt.Errorf("invalid scale: 0")
	}
	return (v - tag.Offset) / scale, nil
}

func encodeHoldingTagNumericValue(tag TagDef, numericValue float64) ([]uint16, error) {
	raw, err := removeScaleOffset(numericValue, tag)
	if err != nil {
		return nil, err
	}

	var words []uint16
	switch tag.DataType {
	case "", "uint16":
		v := math.Round(raw)
		if v < 0 || v > 65535 {
			return nil, fmt.Errorf("value out of uint16 range")
		}
		words = []uint16{uint16(v)}
	case "int16":
		v := math.Round(raw)
		if v < -32768 || v > 32767 {
			return nil, fmt.Errorf("value out of int16 range")
		}
		words = []uint16{uint16(int16(v))}
	case "uint32":
		v := math.Round(raw)
		if v < 0 || v > 4294967295 {
			return nil, fmt.Errorf("value out of uint32 range")
		}
		u := uint32(v)
		words = []uint16{uint16(u >> 16), uint16(u & 0xFFFF)}
	case "int32":
		v := math.Round(raw)
		if v < -2147483648 || v > 2147483647 {
			return nil, fmt.Errorf("value out of int32 range")
		}
		u := uint32(int32(v))
		words = []uint16{uint16(u >> 16), uint16(u & 0xFFFF)}
	case "float32":
		u := math.Float32bits(float32(raw))
		words = []uint16{uint16(u >> 16), uint16(u & 0xFFFF)}
	default:
		return nil, fmt.Errorf("numeric write is not supported for data_type %q", tag.DataType)
	}

	if len(words) == 2 && tag.WordOrder == "lsw" {
		words[0], words[1] = words[1], words[0]
	}
	for i := range words {
		words[i] = applyByteOrder(words[i], tag.ByteOrder)
	}

	return words, nil
}

func encodeHoldingTagStringValue(tag TagDef, s string) ([]uint16, error) {
	if tag.DataType != "string" {
		return nil, fmt.Errorf("string write is only supported for data_type=string")
	}
	if tag.Quantity == 0 {
		return nil, fmt.Errorf("tag quantity must be greater than 0")
	}

	maxBytes := int(tag.Quantity) * 2
	raw := []byte(s)
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("string too long for tag quantity %d (max %d bytes)", tag.Quantity, maxBytes)
	}

	buf := make([]byte, maxBytes)
	copy(buf, raw)
	words := make([]uint16, tag.Quantity)
	for i := 0; i < int(tag.Quantity); i++ {
		w := uint16(buf[i*2])<<8 | uint16(buf[i*2+1])
		words[i] = applyByteOrder(w, tag.ByteOrder)
	}

	return words, nil
}
