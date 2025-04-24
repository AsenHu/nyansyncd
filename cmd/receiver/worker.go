package main

import (
	"io"
	"net"

	"github.com/rs/zerolog/log"
)

const (
	CONTROL uint8 = 0 // 控制流
	DATA    uint8 = 1 // 数据流
)

func handleConn(conn net.Conn) {
	defer conn.Close()
	// 读取第一个字节
	var b [1]byte
	_, err := conn.Read(b[:])
	if err != nil {
		log.Error().Err(err).Msg("Failed to read first byte")
		return
	}
	if b[0] == CONTROL {
		handleControl(conn)
	} else if b[0] == DATA {
		handleData(conn)
	} else {
		log.Error().Msg("Invalid first byte")
	}
}

// handleControl 处理控制流
func handleControl(conn net.Conn) {
	/*
		这里一共使用两个协程处理任务，一个协程专门接受数据，一个协程检查数据后发送数据
		接受数据的协程收到数据后，使用 chan 发送数据到检查数据的协程，这里的 chan 是无缓冲的
		以便让接受数据和处理数据的协程同步，但又不会让磁盘和网络 IO 互相阻塞
		当 chan 关闭时，由检查数据的协程处理完数据后关闭连接
	*/

	// 这里使用一个无缓冲的 chan 来传递 SHA1
	var sha1Chan chan [20]byte
	defer close(sha1Chan)

	// 开启一个协程来处理数据
	go func() {
		defer conn.Close()
		for {
			// 从 chan 中读取 SHA1
			sha1, ok := <-sha1Chan
			if !ok {
				log.Info().Msg("SHA1 channel closed")
				return
			}

			// 检查 SHA1 是否存在
			if !sha1Exists(sha1) {
				// 如果不存在，则发送 SHA1
				_, err := conn.Write(sha1[:])
				if err != nil {
					log.Error().Err(err).Msg("Failed to write SHA1")
					conn.Close()
				}
			}
		}
	}()

	for {
		// 读取 SHA1
		var sha1 [20]byte
		_, err := conn.Read(sha1[:])
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Connection closed by client")
				return
			}
			log.Error().Err(err).Msg("Failed to read SHA1")
			return
		}

		// 将 SHA1 发送到 chan 中
		sha1Chan <- sha1
	}
}

// handleData 处理数据流
func handleData(conn net.Conn) {
	// 等待实现
	conn.Close()
}
