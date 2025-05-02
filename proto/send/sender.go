package send

import "github.com/AsenHu/nyansyncd/proto"

/*
这个方法接受一个文件信息，并将其发送到对方

文件结构如下
SHA1 20 字节
大小 4 字节
宽度 2 字节
高度 2 字节
类型 1 字节
读取时间与建立连接的时间差 2 字节
恶臭的东西 [114] 1 字节
文件内容
*/

func (c *Conn) SendFile(file proto.FileInfo) error {
	// 构建 header
	var header [32]byte
	// SHA1
	copy(header[:20], file.SHA1[:])
	// 文件大小
	header[20] = byte(file.Size >> 24)
	header[21] = byte(file.Size >> 16)
	header[22] = byte(file.Size >> 8)
	header[23] = byte(file.Size)
	// 文件宽度
	header[24] = byte(file.Width >> 8)
	header[25] = byte(file.Width)
	// 文件高度
	header[26] = byte(file.Height >> 8)
	header[27] = byte(file.Height)
	// 文件类型
	header[28] = file.Type
	// 恶臭的东西
	header[31] = 114

	/* 计算时间差
	这个值必须是一个正数
	因此，如果文件的修改时间比连接的时间还要大，则使用 0

	其他情况下，计算连接建立时间与文件修改时间的差值
	然后将差值除 16 分钟，得出一个整数
	然后将这个整数转换为 2 字节的无符号整数
	以表示接近 2 年的时间差
	*/
	diff := (file.Mtime.Unix() - c.time) / (16 * 60)
	switch {
	case diff < 0:
		header[29] = 0
		header[30] = 0
	case diff > 0xFFFF:
		header[29] = 0xFF
		header[30] = 0xFF
	default:
		header[29] = byte(diff >> 8)
		header[30] = byte(diff)
	}

	// 发送 header 和文件内容
	_, err := c.conn.Write(header[:])
	if err != nil {
		return err
	}
	_, err = c.conn.Write(file.File)
	if err != nil {
		return err
	}

	return nil
}
