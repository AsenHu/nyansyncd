package proto

import "time"

const (
	GIF  uint8 = 0
	JPG  uint8 = 1
	WEBP uint8 = 2
	PNG  uint8 = 3
)

type FileInfo struct {
	SHA1   [20]byte  // 文件的 SHA1
	Size   uint32    // 文件的大小
	Width  uint16    // 文件的宽度
	Height uint16    // 文件的高度
	Type   uint8     // 文件的类型
	Mtime  time.Time // 文件的修改时间
	File   []byte    // 文件的内容
}
