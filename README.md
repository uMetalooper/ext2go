# ext2go

A read-only ext2 filesystem reader written in Go, built for learning ext2 internals.

## What it does

Parses a raw ext2 disk image and lets you inspect its contents — superblock metadata, directory trees, file data, and inode block pointer structures — without mounting the image.

## Building

```bash
go build ./...
```

## Usage

```bash
./ext2go <image> <command> [args]
```

| Command | Description |
|---|---|
| `info` | Filesystem metadata (block size, inode count, UUID, block groups) |
| `ls [path]` | List directory contents (default: `/`) |
| `read <path>` | Print file contents to stdout |
| `stat <path>` | Show inode details and raw block pointer list |

### Examples

```bash
# Show filesystem metadata
./ext2go disk.img info

# List root directory
./ext2go disk.img ls /

# List a subdirectory
./ext2go disk.img ls docs

# Read a file
./ext2go disk.img read docs/notes.txt

# Inspect inode block pointers (useful for large files)
./ext2go disk.img stat docs/bigfile.txt
```

## Creating a test image (macOS)

```bash
# Install e2fsprogs for mke2fs and debugfs
brew install e2fsprogs

# Create a 4MB blank image and format as ext2
dd if=/dev/zero of=disk.img bs=1024 count=4096
/opt/homebrew/opt/e2fsprogs/sbin/mke2fs -t ext2 disk.img

# Write files into the image using debugfs
/opt/homebrew/opt/e2fsprogs/sbin/debugfs -w disk.img <<'EOF'
mkdir docs
write /tmp/hello.txt hello.txt
write /tmp/bigfile.txt docs/bigfile.txt
EOF
```

## How ext2 works

An ext2 disk image is a flat binary file divided into fixed-size **blocks** (typically 1KB–4KB). Blocks are grouped into **block groups**, each containing:

```
Block Group
├── Superblock copy       — filesystem metadata (magic, block size, inode count, ...)
├── Group Descriptor      — where the bitmaps and inode table are
├── Block Bitmap          — one bit per block: 0=free, 1=used
├── Inode Bitmap          — one bit per inode: 0=free, 1=used
├── Inode Table           — array of 128-byte inode structs
└── Data Blocks           — actual file and directory content
```

### Key concepts

**Superblock** — located at byte offset 1024, always. Contains the magic number (`0xef53`), block size, inode count, and other global metadata.

**Inode** — every file and directory is an inode. An inode stores permissions, timestamps, size, and an array of 15 block pointers — but not the filename. Names live in directory entries.

```
inode.Block[0..11]  → 12 direct data blocks         (up to 12KB with 1KB blocks)
inode.Block[12]     → single-indirect block          (points to a block of block numbers)
inode.Block[13]     → double-indirect block          (points to blocks of block numbers)
inode.Block[14]     → triple-indirect block
```

**Directory entry** — a directory's data blocks contain a linked list of variable-length records, each mapping a filename to an inode number. Path resolution walks these records one component at a time.

## Code structure

| File | Responsibility |
|---|---|
| `disk.go` | Raw I/O: `readAt` and `readBlock` over an image file |
| `superblock.go` | Parse the superblock struct; compute block/inode size |
| `group.go` | Parse the block group descriptor table |
| `inode.go` | Parse inodes; resolve the block pointer tree (direct + indirect) |
| `dir.go` | Parse directory entries from a directory inode's data blocks |
| `fs.go` | High-level: mount, path resolution, `ls`, `read`, `stat` |
| `main.go` | CLI entry point |

## Related

- [ext2py](https://github.com/EarlGray/ext2py) — the Python implementation this was ported from
- [Linux kernel ext2.h](https://github.com/torvalds/linux/blob/master/fs/ext2/ext2.h) — authoritative on-disk struct definitions
- [The Second Extended Filesystem](https://www.nongnu.org/ext2-doc/ext2.html) — full specification
