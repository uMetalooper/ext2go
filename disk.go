package main

import (
	"os"
	"sync"
)

type disk struct {
	f     *os.File
	blksz int
	mu    sync.Mutex
}

func openDisk(path string) (*disk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &disk{f: f}, nil
}

func (d *disk) close() {
	d.f.Close()
}

func (d *disk) readAt(buf []byte, offset int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.f.ReadAt(buf, offset)
	return err
}

func (d *disk) readBlock(blockNum int) ([]byte, error) {
	buf := make([]byte, d.blksz)
	err := d.readAt(buf, int64(blockNum)*int64(d.blksz))
	return buf, err
}
