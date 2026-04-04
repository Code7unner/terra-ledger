package http

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

func decodeBorshString(data []byte, offset int) (string, int, error) {
	if offset+4 > len(data) {
		return "", offset, fmt.Errorf("borsh string: not enough bytes for length at offset %d", offset)
	}
	strLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	if offset+strLen > len(data) {
		return "", offset, fmt.Errorf("borsh string: not enough bytes for string of length %d at offset %d", strLen, offset)
	}
	s := string(data[offset : offset+strLen])
	offset += strLen

	return s, offset, nil
}

func decodePubkey(data []byte, offset int) (string, int, error) {
	if offset+32 > len(data) {
		return "", offset, fmt.Errorf("borsh pubkey: not enough bytes at offset %d", offset)
	}
	pk := solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	return pk.String(), offset, nil
}

func decodeU16LE(data []byte, offset int) (uint16, int, error) {
	if offset+2 > len(data) {
		return 0, offset, fmt.Errorf("borsh u16: not enough bytes at offset %d", offset)
	}
	v := binary.LittleEndian.Uint16(data[offset : offset+2])
	return v, offset + 2, nil
}

func decodeU32LE(data []byte, offset int) (uint32, int, error) {
	if offset+4 > len(data) {
		return 0, offset, fmt.Errorf("borsh u32: not enough bytes at offset %d", offset)
	}
	v := binary.LittleEndian.Uint32(data[offset : offset+4])
	return v, offset + 4, nil
}

func decodeU64LE(data []byte, offset int) (uint64, int, error) {
	if offset+8 > len(data) {
		return 0, offset, fmt.Errorf("borsh u64: not enough bytes at offset %d", offset)
	}
	v := binary.LittleEndian.Uint64(data[offset : offset+8])
	return v, offset + 8, nil
}
