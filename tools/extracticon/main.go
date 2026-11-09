//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	loadLibraryAsDatafile = 0x00000002
	rtIcon                = 3
	rtGroupIcon           = 14
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: extracticon <exe> <out.ico>")
		os.Exit(2)
	}
	data, err := extractGroupIcon(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[2], data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", os.Args[2], len(data))
}

func extractGroupIcon(exe string) ([]byte, error) {
	mod, err := windows.LoadLibraryEx(exe, 0, loadLibraryAsDatafile)
	if err != nil {
		return nil, err
	}
	defer windows.FreeLibrary(mod)

	group := loadResource(mod, 1, rtGroupIcon)
	if group == nil {
		return nil, fmt.Errorf("no icon group in %s", exe)
	}
	if len(group) < 6 {
		return nil, fmt.Errorf("icon group too small")
	}
	count := binary.LittleEndian.Uint16(group[4:6])
	const entrySize = 14
	if len(group) < 6+int(count)*entrySize {
		return nil, fmt.Errorf("icon group truncated")
	}
	images := make([]iconImage, 0, count)
	for i := 0; i < int(count); i++ {
		off := 6 + i*entrySize
		meta := append([]byte(nil), group[off:off+12]...)
		id := binary.LittleEndian.Uint16(group[off+12 : off+14])
		img := loadResource(mod, uintptr(id), rtIcon)
		if img == nil {
			return nil, fmt.Errorf("missing icon id %d", id)
		}
		images = append(images, iconImage{meta: meta, bytes: img})
	}
	return writeICO(images), nil
}

type iconImage struct {
	meta  []byte
	bytes []byte
}

func writeICO(images []iconImage) []byte {
	count := len(images)
	header := make([]byte, 6+count*16)
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], uint16(count))
	offset := 6 + count*16
	var blobs []byte
	for i, img := range images {
		entry := header[6+i*16 : 6+(i+1)*16]
		copy(entry[:12], img.meta[:12])
		binary.LittleEndian.PutUint32(entry[8:12], uint32(len(img.bytes)))
		binary.LittleEndian.PutUint32(entry[12:16], uint32(offset))
		blobs = append(blobs, img.bytes...)
		offset += len(img.bytes)
	}
	return append(header, blobs...)
}

func loadResource(mod windows.Handle, name, kind uintptr) []byte {
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	find := k32.NewProc("FindResourceW")
	load := k32.NewProc("LoadResource")
	lock := k32.NewProc("LockResource")
	size := k32.NewProc("SizeofResource")
	res, _, _ := find.Call(uintptr(mod), name, kind)
	if res == 0 {
		return nil
	}
	loaded, _, _ := load.Call(uintptr(mod), res)
	if loaded == 0 {
		return nil
	}
	ptr, _, _ := lock.Call(loaded)
	if ptr == 0 {
		return nil
	}
	n, _, _ := size.Call(uintptr(mod), res)
	if n == 0 {
		return nil
	}
	return append([]byte(nil), unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&ptr))), n)...)
}
