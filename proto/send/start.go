package send

import (
	"net"
	"time"
)

/*
这个文件处理建立连接的工作

它会准备一个 header 放入连接中，并准备好解析数据的 Reader
然后返回一个 Conn 对象

上层通过 Chan 获取到对方的文件列表，然后通过调用方法来发送文件
*/

/*
协议头是这样的
自己目录里的第一个文件的 SHA1 20 字节
时间戳 8 字节
*/

func NewConn(sha1 [20]byte, conn net.TCPConn) *Conn {
	// 构建要发送的头部
	time := time.Now().Unix()
	var header [28]byte
	copy(header[:20], sha1[:])
	header[20] = byte(time >> 56)
	header[21] = byte(time >> 48)
	header[22] = byte(time >> 40)
	header[23] = byte(time >> 32)
	header[24] = byte(time >> 24)
	header[25] = byte(time >> 16)
	header[26] = byte(time >> 8)
	header[27] = byte(time)

	// 发送头部
	conn.Write(header[:])

	// 创建一个 Conn 对象
	c := &Conn{
		conn:     conn,
		time:     time,
		recFiles: make(chan [20]byte, 100),
	}

	// 启动一个协程，读取对方的文件列表
	go c.reader()
	return c
}
