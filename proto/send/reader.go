package send

/*
这个方法在连接建立后，从连接中读取数据，并塞到 Chan 中
当对方关闭连接时，关闭 Chan
*/

func (c *Conn) reader() {
	for {
		// 读取数据
		var sha1 [20]byte
		n, err := c.conn.Read(sha1[:])
		if err != nil {
			break
		}
		if n != 20 {
			break
		}
		c.recFiles <- sha1
	}
	// 关闭连接读取方向和 Chan
	c.conn.CloseRead()
	close(c.recFiles)
}

// 读取对方的文件列表
func (c *Conn) GetFile() *[20]byte {
	// 从 Chan 中读取数据
	sha1, ok := <-c.recFiles
	if !ok {
		return nil
	}
	return &sha1
}
