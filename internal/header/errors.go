package header

import "errors"

/*
这里定义了检查头部信息的错误类型
头部信息的错误类型包括：
- ErrInvalidType
- ErrInvalidWidth
- ErrInvalidHeight
- ErrInvalidSize
- ErrInvalidTime
- ErrInvalidHash
*/

var (
	// ErrInvalidType 不合法的文件类型
	ErrInvalidType = errors.New("unknown file type")
	// ErrInvalidWidth 错误宽度
	ErrInvalidWidth = errors.New("the width is not in the range of 100-20000")
	// ErrInvalidHeight 错误高度
	ErrInvalidHeight = errors.New("the height is not in the range of 100-20000")
	// ErrInvalidSize 错误大小
	ErrInvalidSize = errors.New("the size is not in the range of 10MB(GIF) 20MB(JPG, WEBP) 50MB(PNG)")
	// ErrInvalidTime 错误时间
	ErrInvalidTime = errors.New("the time is from the future")
	// ErrInvalidHash 错误的 hash 值
	ErrInvalidHash = errors.New("the first 40 characters of the file name is not a hash")
)
