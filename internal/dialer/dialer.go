package dialer

import (
	"net"
	"time"
)

/*
这个包提供一个对 TCP 连接的封装
它提供 io.Writer 给控制流
以及一个 io.Reader 的控制流和 io.Writer 的数据流

当底层连接断开时，阻塞上层的操作，直到成功
*/

type Dialer struct {
	dial string
	ctrl net.Conn
	data net.Conn
}

func NewDialer(dial string) *Dialer {
	return &Dialer{
		dial: dial,
		ctrl: newConn(dial, 0),
		data: newConn(dial, 1),
	}
}

func newConn(dial string, p uint8) net.Conn {
	for {
		conn, err := net.Dial("tcp", dial)
		if err != nil {
			time.Sleep(time.Second * 16)
			continue
		}
		// 发送第一个字节
		_, err = conn.Write([]byte{p})
		if err != nil {
			conn.Close()
			time.Sleep(time.Second * 16)
			continue
		}
		return conn
	}
}

func (d *Dialer) CtrlW(p []byte) {
	for {
		_, err := d.ctrl.Write(p)
		if err == nil {
			break
		}
		d.ctrl.Close()
		d.ctrl = newConn(d.dial, 0)
	}
}

func (d *Dialer) CtrlR(p []byte) {
	for {
		_, err := d.ctrl.Read(p)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 16)
	}
}

func (d *Dialer) DataW(p []byte) {
	for {
		_, err := d.data.Write(p)
		if err == nil {
			break
		}
		d.data.Close()
		d.data = newConn(d.dial, 1)
	}
}
