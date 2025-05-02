package send

import "net"

/*
这个包指示协议的头部
这个协议工作的流程是这样的

1. sender 打开一个 TCP 连接
2. sender 请求对方列出一个哈希分片中的文件
3. receiver 返回一个哈希分片中的文件列表
4. sender 根据对方的文件列表，发送对方没有的文件
*/

type Conn struct {
	conn     net.TCPConn   // 连接
	time     int64         // 连接的开始时间
	recFiles chan [20]byte // 对方已有的文件列表
}
