package main

import (
	"fmt"
	"strings"
)

type ext2FS struct {
	disk   *disk
	sb     *superblock
	groups []groupDesc
	root   *inode
}

func mount(imagePath string) (*ext2FS, error) {
	d, err := openDisk(imagePath)
	if err != nil {
		return nil, err
	}

	sb, err := readSuperblock(d)
	if err != nil {
		d.close()
		return nil, err
	}
	d.blksz = sb.BlockSize

	groups, err := readGroupDescs(d, sb)
	if err != nil {
		d.close()
		return nil, err
	}

	root, err := readInode(d, sb, groups, rootInodeNum)
	if err != nil {
		d.close()
		return nil, err
	}

	return &ext2FS{disk: d, sb: sb, groups: groups, root: root}, nil
}

func (fs *ext2FS) umount() {
	fs.disk.close()
}

func (fs *ext2FS) inode(num int) (*inode, error) {
	return readInode(fs.disk, fs.sb, fs.groups, num)
}

// inodeByPath resolves an absolute or relative path to an inode by
// walking directory entries one component at a time from the root.
func (fs *ext2FS) inodeByPath(path string) (*inode, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return fs.root, nil
	}

	cur := fs.root
	for _, part := range strings.Split(path, "/") {
		if part == "" {
			continue
		}
		entries, err := readDir(fs.disk, fs.sb, cur)
		if err != nil {
			return nil, err
		}
		var found *inode
		for _, e := range entries {
			if e.Name == part {
				found, err = fs.inode(int(e.Inode))
				if err != nil {
					return nil, err
				}
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("%q: no such file or directory", part)
		}
		cur = found
	}
	return cur, nil
}

func (fs *ext2FS) ls(path string) error {
	ino, err := fs.inodeByPath(path)
	if err != nil {
		return err
	}

	if !ino.IsDir() {
		fmt.Printf("%s %4d %4d %8d %s\n",
			ino.modeStr(), ino.UID, ino.GID, ino.Size, formatTime(ino.MTime))
		return nil
	}

	entries, err := readDir(fs.disk, fs.sb, ino)
	if err != nil {
		return err
	}
	for _, e := range entries {
		child, err := fs.inode(int(e.Inode))
		if err != nil {
			return err
		}
		fmt.Printf("%s %3d %4d:%-4d %8d  %s  %s\n",
			child.modeStr(), child.LinksCount, child.UID, child.GID,
			child.Size, formatTime(child.MTime), e.Name)
	}
	return nil
}

func (fs *ext2FS) readFile(path string) ([]byte, error) {
	ino, err := fs.inodeByPath(path)
	if err != nil {
		return nil, err
	}
	if ino.IsDir() {
		return nil, fmt.Errorf("%s: is a directory", path)
	}

	data := make([]byte, 0, ino.Size)
	remaining := int(ino.Size)
	for _, blockNum := range ino.blockList {
		if remaining <= 0 {
			break
		}
		block, err := fs.disk.readBlock(int(blockNum))
		if err != nil {
			return nil, err
		}
		toRead := fs.sb.BlockSize
		if toRead > remaining {
			toRead = remaining
		}
		data = append(data, block[:toRead]...)
		remaining -= toRead
	}
	return data, nil
}

func (fs *ext2FS) stat(path string) error {
	ino, err := fs.inodeByPath(path)
	if err != nil {
		return err
	}
	fmt.Printf("Inode     : %d\n", ino.num)
	fmt.Printf("Mode      : %s (0%o)\n", ino.modeStr(), ino.Mode&0x1FF)
	fmt.Printf("UID/GID   : %d / %d\n", ino.UID, ino.GID)
	fmt.Printf("Size      : %d bytes\n", ino.Size)
	fmt.Printf("Links     : %d\n", ino.LinksCount)
	fmt.Printf("Accessed  : %s\n", formatTime(ino.ATime))
	fmt.Printf("Created   : %s\n", formatTime(ino.CTime))
	fmt.Printf("Modified  : %s\n", formatTime(ino.MTime))
	fmt.Printf("Blocks    : %d (direct), single-indirect=%d, double=%d, triple=%d\n",
		countNonZero(ino.blockPtrs[:nDirBlocks]),
		ino.blockPtrs[singleIndirect],
		ino.blockPtrs[doubleIndirect],
		ino.blockPtrs[tripleIndirect])
	fmt.Printf("Block list: %v\n", ino.blockList)
	return nil
}

func countNonZero(ptrs []uint32) int {
	n := 0
	for _, p := range ptrs {
		if p != 0 {
			n++
		}
	}
	return n
}
