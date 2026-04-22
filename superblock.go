package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	superblockOffset = 1024
	ext2Magic        = 0xef53
	rootInodeNum     = 2
)

// superblockRaw mirrors the on-disk ext2 superblock layout exactly.
// Fields are read sequentially by encoding/binary with no padding.
type superblockRaw struct {
	InodesCount     uint32
	BlocksCount     uint32
	RBlocksCount    uint32
	FreeBlocksCount uint32
	FreeInodesCount uint32
	FirstDataBlock  uint32
	LogBlockSize    uint32 // block size = 1024 << LogBlockSize
	LogFragSize     uint32
	BlocksPerGroup  uint32
	FragsPerGroup   uint32
	InodesPerGroup  uint32
	Mtime           uint32
	Wtime           uint32
	MntCount        uint16
	MaxMntCount     uint16
	Magic           uint16 // must be 0xef53
	State           uint16
	Errors          uint16
	MinorRevLevel   uint16
	LastCheck       uint32
	CheckInterval   uint32
	CreatorOS       uint32
	RevLevel        uint32
	DefResUID       uint16
	DefResGID       uint16
	// rev1+ fields
	FirstIno        uint32
	InodeSize       uint16
	BlockGroupNr    uint16
	FeatureCompat   uint32
	FeatureIncompat uint32
	FeatureROCompat uint32
	UUID            [16]byte
	VolumeName      [16]byte
	LastMounted     [64]byte
	AlgoBitmap      uint32
	PreallocBlocks  uint8
	PreallocDirBlks uint8
}

type superblock struct {
	raw            superblockRaw
	BlockSize      int
	InodeSize      int
	BlocksPerGroup uint32
	InodesPerGroup uint32
	FirstDataBlock uint32
	InodesCount    uint32
	BlocksCount    uint32
	FreeInodes     uint32
	FreeBlocks     uint32
	VolumeName     string
	UUID           string
}

func readSuperblock(d *disk) (*superblock, error) {
	buf := make([]byte, 1024)
	if err := d.readAt(buf, superblockOffset); err != nil {
		return nil, err
	}

	var raw superblockRaw
	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, &raw); err != nil {
		return nil, err
	}

	if raw.Magic != ext2Magic {
		return nil, fmt.Errorf("invalid ext2 magic: 0x%x (expected 0x%x)", raw.Magic, ext2Magic)
	}

	sb := &superblock{
		raw:            raw,
		BlockSize:      1024 << raw.LogBlockSize,
		BlocksPerGroup: raw.BlocksPerGroup,
		InodesPerGroup: raw.InodesPerGroup,
		FirstDataBlock: raw.FirstDataBlock,
		InodesCount:    raw.InodesCount,
		BlocksCount:    raw.BlocksCount,
		FreeInodes:     raw.FreeInodesCount,
		FreeBlocks:     raw.FreeBlocksCount,
		VolumeName:     strings.TrimRight(string(raw.VolumeName[:]), "\x00"),
		UUID:           formatUUID(raw.UUID),
	}

	if raw.RevLevel > 0 {
		sb.InodeSize = int(raw.InodeSize)
	} else {
		sb.InodeSize = 128
	}

	return sb, nil
}

func formatUUID(b [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func formatTime(t uint32) string {
	return time.Unix(int64(t), 0).Format("2006-01-02 15:04:05")
}
