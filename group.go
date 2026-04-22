package main

import (
	"bytes"
	"encoding/binary"
)

const groupDescSize = 32

// groupDescRaw mirrors the on-disk block group descriptor (32 bytes).
type groupDescRaw struct {
	BlockBitmap     uint32
	InodeBitmap     uint32
	InodeTable      uint32
	FreeBlocksCount uint16
	FreeInodesCount uint16
	UsedDirsCount   uint16
	Pad             uint16
	Reserved        [12]byte
}

type groupDesc struct {
	BlockBitmap uint32
	InodeBitmap uint32
	InodeTable  uint32
	FreeBlocks  uint16
	FreeInodes  uint16
}

func readGroupDescs(d *disk, sb *superblock) ([]groupDesc, error) {
	// GDT is always in the block immediately after the superblock's block.
	// FirstDataBlock=1 for 1KB blocks (superblock at block 1), GDT at block 2.
	// FirstDataBlock=0 for >=2KB blocks (superblock at block 0), GDT at block 1.
	gdtBlock := int(sb.FirstDataBlock) + 1

	numGroups := (int(sb.BlocksCount) + int(sb.BlocksPerGroup) - 1) / int(sb.BlocksPerGroup)

	data, err := d.readBlock(gdtBlock)
	if err != nil {
		return nil, err
	}

	groups := make([]groupDesc, numGroups)
	for i := 0; i < numGroups; i++ {
		var raw groupDescRaw
		if err := binary.Read(bytes.NewReader(data[i*groupDescSize:]), binary.LittleEndian, &raw); err != nil {
			return nil, err
		}
		groups[i] = groupDesc{
			BlockBitmap: raw.BlockBitmap,
			InodeBitmap: raw.InodeBitmap,
			InodeTable:  raw.InodeTable,
			FreeBlocks:  raw.FreeBlocksCount,
			FreeInodes:  raw.FreeInodesCount,
		}
	}
	return groups, nil
}
