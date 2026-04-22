package main

import (
	"encoding/binary"
	"bytes"
)

const dirEntryHeaderSize = 8

// dirEntryRaw is the fixed-size header of each directory entry.
// The variable-length name immediately follows in the block data.
type dirEntryRaw struct {
	Inode    uint32
	RecLen   uint16
	NameLen  uint8
	FileType uint8
}

type dirEntry struct {
	Inode    uint32
	Name     string
	FileType uint8
}

// readDir reads all directory entries from a directory inode.
// A directory is just a file whose content is packed dirEntry records.
func readDir(d *disk, sb *superblock, ino *inode) ([]dirEntry, error) {
	var entries []dirEntry

	for _, blockNum := range ino.blockList {
		data, err := d.readBlock(int(blockNum))
		if err != nil {
			return nil, err
		}

		offset := 0
		for offset < sb.BlockSize {
			var raw dirEntryRaw
			if err := binary.Read(bytes.NewReader(data[offset:]), binary.LittleEndian, &raw); err != nil {
				break
			}
			// inode=0 means deleted/unused entry; RecLen=0 would loop forever
			if raw.Inode == 0 || raw.RecLen == 0 {
				break
			}

			nameStart := offset + dirEntryHeaderSize
			nameEnd := nameStart + int(raw.NameLen)
			if nameEnd > len(data) {
				break
			}

			entries = append(entries, dirEntry{
				Inode:    raw.Inode,
				Name:     string(data[nameStart:nameEnd]),
				FileType: raw.FileType,
			})

			offset += int(raw.RecLen)
		}
	}

	return entries, nil
}
