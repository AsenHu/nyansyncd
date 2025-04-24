package header

import "time"

/*
这里定义了每个图片帧的头部信息
每个图片帧的头部信息包括：
- SHA1
- Size 最大 50 兆字节
- Width
- Height
- Type
- Time
*/

type ImageHeader struct {
	SHA1   [20]byte  // SHA1
	Size   uint32    // Size 最大 50 兆字节
	Width  uint16    // Width
	Height uint16    // Height
	Type   uint8     // Type
	Time   time.Time // Time
}

/*
在图片帧的头部信息里，格式如下
SHA1: 20 bytes
Size: 4 bytes
Width: 2 bytes
Height: 2 bytes
Type: 1 byte
Time: 3 bytes

宽度和高度不得小于 100, 不得大于 20000

其中，Type 的值如下：
- 0: GIF 最大 10 兆字节
- 1: JPG 最大 20 兆字节
- 2: WEBP 最大 20 兆字节
- 3: PNG 最大 50 兆字节

时间并不是时间戳，而是当前时间的过去的 1/4096 天 * Time 的值，0 表示当前时间
例如现在的时间戳是 86400
当 Time 的值为 0 时，就是 86400
当 Time 的值为 1 时，就是 86400 秒 - 1/4096 天 = 86378.90625 秒
当 Time 的值为 2 时，就是 86400 秒 - 2/4096 天 = 86357.8125 秒
*/

func (i *ImageHeader) Check() error {
	// 检查 Type 是否合法
	if i.Type > 3 {
		return ErrInvalidType
	}

	// 检查宽度和高度是否合法
	if i.Width < 100 || i.Width > 20000 {
		return ErrInvalidWidth
	}
	if i.Height < 100 || i.Height > 20000 {
		return ErrInvalidHeight
	}

	// 检查大小是否合法
	switch i.Type {
	case 0:
		if i.Size > 10*1024*1024 {
			return ErrInvalidSize
		}
	case 1:
		if i.Size > 20*1024*1024 {
			return ErrInvalidSize
		}
	case 2:
		if i.Size > 20*1024*1024 {
			return ErrInvalidSize
		}
	case 3:
		if i.Size > 50*1024*1024 {
			return ErrInvalidSize
		}
	}

	// 检查时间是否合法
	// 如果比当前时间还大，说明不合法
	if i.Time.Unix() > time.Now().Unix() {
		return ErrInvalidTime
	}

	return nil
}

// 将信息转换为字节数组
func (i *ImageHeader) ToBytes() (result [32]byte, err error) {
	// 检查合法性
	if err := i.Check(); err != nil {
		return [32]byte{}, err
	}

	// 计算时间差
	timeDiff := time.Now().Unix() - i.Time.Unix()
	timeByteDiff := uint32(timeDiff * 4096 / 86400)

	// 复制 SHA1
	copy(result[:20], i.SHA1[:])
	// 复制其他信息
	result[20] = byte(i.Size >> 24)
	result[21] = byte(i.Size >> 16)
	result[22] = byte(i.Size >> 8)
	result[23] = byte(i.Size)
	result[24] = byte(i.Width >> 8)
	result[25] = byte(i.Width)
	result[26] = byte(i.Height >> 8)
	result[27] = byte(i.Height)
	result[28] = i.Type
	result[29] = byte(timeByteDiff >> 16)
	result[30] = byte(timeByteDiff >> 8)
	result[31] = byte(timeByteDiff)

	// 返回结果
	return result, nil
}

// 从字节数组中读取信息
func ToStruct(data [32]byte) (i ImageHeader, err error) {
	i = ImageHeader{
		SHA1:   ([20]byte)(data[0:20]),
		Size:   uint32(data[20])<<24 | uint32(data[21])<<16 | uint32(data[22])<<8 | uint32(data[23]),
		Width:  uint16(data[24])<<8 | uint16(data[25]),
		Height: uint16(data[26])<<8 | uint16(data[27]),
		Type:   data[28],
		Time:   time.Now().Add(-time.Duration(uint32(data[29])<<16|uint32(data[30])<<8|uint32(data[31])) * 86400 / 4096),
	}

	// 检查合法性
	if err := i.Check(); err != nil {
		// 如果是时间不合法，仍然返回结果
		if err == ErrInvalidTime {
			return i, err
		}
		// 其他错误，返回空值
		return ImageHeader{}, err
	}

	// 返回结果
	return i, nil
}
