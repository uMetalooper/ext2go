package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	nDirBlocks     = 12 // number of direct block pointers in an inode
	singleIndirect = 12 // index of single-indirect pointer in Block array
	doubleIndirect = 13
	tripleIndirect = 14
)

// inodeRaw mirrors the on-disk inode structure (128 bytes for rev0/rev1).
type inodeRaw struct {
	Mode       uint16
	UID        uint16
	Size       uint32
	ATime      uint32
	CTime      uint32
	MTime      uint32
	DTime      uint32
	GID        uint16
	LinksCount uint16
	Blocks     uint32 // 512-byte block count (not fs-block count)
	Flags      uint32
	OSSpec1    uint32
	Block      [15]uint32 // [0..11]=direct, [12]=single, [13]=double, [14]=triple indirect
	Generation uint32
	FileACL    uint32
	DirACL     uint32
	FAddr      uint32
	OSSpec2    [12]byte
}

type inode struct {
	num        int
	Mode       uint16
	UID        uint16
	GID        uint16
	Size       uint32
	ATime      uint32
	CTime      uint32
	MTime      uint32
	LinksCount uint16
	blockPtrs  [15]uint32
	blockList  []uint32
}

func (ino *inode) IsDir() bool  { return ino.Mode&0xF000 == 0x4000 }
func (ino *inode) IsFile() bool { return ino.Mode&0xF000 == 0x8000 }
func (ino *inode) IsLink() bool { return ino.Mode&0xF000 == 0xA000 }

func (ino *inode) modeStr() string {
	typeChar := '-'
	switch {
	case ino.IsDir():
		typeChar = 'd'
	case ino.IsLink():
		typeChar = 'l'
	}
	bits := []struct {
		bit  uint16
		char byte
	}{
		{0400, 'r'}, {0200, 'w'}, {0100, 'x'},
		{0040, 'r'}, {0020, 'w'}, {0010, 'x'},
		{0004, 'r'}, {0002, 'w'}, {0001, 'x'},
	}
	perm := make([]byte, 9)
	for i, b := range bits {
		if ino.Mode&b.bit != 0 {
			perm[i] = b.char
		} else {
			perm[i] = '-'
		}
	}
	return string(typeChar) + string(perm)
}

func readInode(d *disk, sb *superblock, groups []groupDesc, inodeNum int) (*inode, error) {
	groupIdx := (inodeNum - 1) / int(sb.InodesPerGroup)
	localIdx := (inodeNum - 1) % int(sb.InodesPerGroup)

	if groupIdx >= len(groups) {
		return nil, fmt.Errorf("inode %d: group index %d out of range", inodeNum, groupIdx)
	}

	offset := int64(groups[groupIdx].InodeTable)*int64(sb.BlockSize) + int64(localIdx)*int64(sb.InodeSize)

	buf := make([]byte, sb.InodeSize)
	if err := d.readAt(buf, offset); err != nil {
		return nil, err
	}

	var raw inodeRaw
	if err := binary.Read(bytes.NewReader(buf), binary.LittleEndian, &raw); err != nil {
		return nil, err
	}

	ino := &inode{
		num:        inodeNum,
		Mode:       raw.Mode,
		UID:        raw.UID,
		GID:        raw.GID,
		Size:       raw.Size,
		ATime:      raw.ATime,
		CTime:      raw.CTime,
		MTime:      raw.MTime,
		LinksCount: raw.LinksCount,
		blockPtrs:  raw.Block,
	}

	// Short symlinks store the target in the block pointer array itself (<=60 chars).
	// They have no real block list to build.
	isShortLink := ino.IsLink() && ino.Size <= uint32(nDirBlocks*4)
	if !isShortLink {
		var err error
		ino.blockList, err = buildBlockList(d, sb, raw.Block)
		if err != nil {
			return nil, fmt.Errorf("inode %d build block list: %w", inodeNum, err)
		}
	}

	return ino, nil
}

func buildBlockList(d *disk, sb *superblock, ptrs [15]uint32) ([]uint32, error) {
	var blocks []uint32

	// Direct blocks (indices 0–11)
	for i := 0; i < nDirBlocks; i++ {
		if ptrs[i] == 0 {
			return blocks, nil
		}
		blocks = append(blocks, ptrs[i])
	}

	// Single-indirect: ptrs[12] points to a block full of block numbers
	if ptrs[singleIndirect] != 0 {
		indirect, err := readPtrBlock(d, sb, ptrs[singleIndirect])
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, indirect...)
	}

	// Double-indirect: ptrs[13] → block of pointers → each points to a block of block numbers
	if ptrs[doubleIndirect] != 0 {
		l1, err := readPtrBlock(d, sb, ptrs[doubleIndirect])
		if err != nil {
			return nil, err
		}
		for _, ptr := range l1 {
			l2, err := readPtrBlock(d, sb, ptr)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, l2...)
		}
	}

	// Triple-indirect: ptrs[14] → block → block → block of block numbers
	if ptrs[tripleIndirect] != 0 {
		l1, err := readPtrBlock(d, sb, ptrs[tripleIndirect])
		if err != nil {
			return nil, err
		}
		for _, ptr1 := range l1 {
			l2, err := readPtrBlock(d, sb, ptr1)
			if err != nil {
				return nil, err
			}
			for _, ptr2 := range l2 {
				l3, err := readPtrBlock(d, sb, ptr2)
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, l3...)
			}
		}
	}

	return blocks, nil
}

// readPtrBlock reads a block and returns the non-zero uint32 values as a slice of block numbers.
func readPtrBlock(d *disk, sb *superblock, blockNum uint32) ([]uint32, error) {
	data, err := d.readBlock(int(blockNum))
	if err != nil {
		return nil, err
	}

	ptrsPerBlock := sb.BlockSize / 4
	ptrs := make([]uint32, 0, ptrsPerBlock)
	r := bytes.NewReader(data)
	for i := 0; i < ptrsPerBlock; i++ {
		var p uint32
		if err := binary.Read(r, binary.LittleEndian, &p); err != nil {
			break
		}
		if p == 0 {
			break
		}
		ptrs = append(ptrs, p)
	}
	return ptrs, nil
}
