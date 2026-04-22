package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	imgPath := os.Args[1]
	cmd := os.Args[2]

	fs, err := mount(imgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error mounting %s: %v\n", imgPath, err)
		os.Exit(1)
	}
	defer fs.umount()

	switch cmd {
	case "info":
		printInfo(fs)

	case "ls":
		path := "/"
		if len(os.Args) > 3 {
			path = os.Args[3]
		}
		if err := fs.ls(path); err != nil {
			fmt.Fprintf(os.Stderr, "ls: %v\n", err)
			os.Exit(1)
		}

	case "read":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: ext2go <image> read <path>")
			os.Exit(1)
		}
		data, err := fs.readFile(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "read: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(data)

	case "stat":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: ext2go <image> stat <path>")
			os.Exit(1)
		}
		if err := fs.stat(os.Args[3]); err != nil {
			fmt.Fprintf(os.Stderr, "stat: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func printInfo(fs *ext2FS) {
	sb := fs.sb
	fmt.Printf("UUID         : %s\n", sb.UUID)
	fmt.Printf("Volume name  : %q\n", sb.VolumeName)
	fmt.Printf("Block size   : %d bytes\n", sb.BlockSize)
	fmt.Printf("Inode size   : %d bytes\n", sb.InodeSize)
	fmt.Printf("Total blocks : %d\n", sb.BlocksCount)
	fmt.Printf("Free blocks  : %d\n", sb.FreeBlocks)
	fmt.Printf("Total inodes : %d\n", sb.InodesCount)
	fmt.Printf("Free inodes  : %d\n", sb.FreeInodes)
	fmt.Printf("Block groups : %d\n", len(fs.groups))
	for i, g := range fs.groups {
		fmt.Printf("  group[%d]: block_bitmap=%d inode_bitmap=%d inode_table=%d free_blocks=%d free_inodes=%d\n",
			i, g.BlockBitmap, g.InodeBitmap, g.InodeTable, g.FreeBlocks, g.FreeInodes)
	}
}

func usage() {
	fmt.Println("usage: ext2go <image> <command> [args]")
	fmt.Println("commands:")
	fmt.Println("  info           show filesystem metadata")
	fmt.Println("  ls   [path]    list directory (default: /)")
	fmt.Println("  read <path>    print file contents")
	fmt.Println("  stat <path>    show inode details and block list")
}
