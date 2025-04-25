package main

import (
	"flag"
	"net"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

/*
在整个协议中，一共有两个流
一个是控制流，一个是数据流

两个流都是 TCP 连接，并且是由 sender 先连接 receiver，因为 receiver 一定能监听一个 TCP 端口（因为它是一个服务端）

对于一个图片来说，它是这样被传输的
1. sender 先连接 receiver 的控制流
2. sender 通过控制流发送图片的 SHA1
3. receiver 收到 SHA1 后，检查是否已经存在
4. receiver 通过控制流发送不存在的 SHA1
5. sender 收到不存在的 SHA1 后，通过数据流发送图片的头部和数据
6. receiver 收到图片的头部和数据后，检查头部是否合法
7. receiver 收到合法的数据后，保存图片，这个图片传输完成
8. receiver 收到不合法的数据后，将图片删除，并且通过数据流返回错误图片的 SHA1
9. sender 收到错误的 SHA1 后，重新传输图片

当 sender 发起连接时，如果是控制流，则第一个字节是 0，如果是数据流，则第一个字节是 1
当 receiver 收到连接时，如果第一个字节是 0，则是控制流，之后每 20 个字节是一个 SHA1
receiver 从控制流返回 SHA1 时，无需返回任何头部，直接每 20 个字节是一个 SHA1

当 receiver 收到连接时，如果第一个字节是 1，则是数据流，之后依次是 32 个字节的头部数据，和可变长度的数据
从数据流返回 SHA1 时，返回 20 个字节的 SHA1，也是无需返回任何头部
*/

var (
	LISTEN     string // 监听的地址
	CACHE_PATH string // 缓存的路径
)

func init() {
	flag.StringVar(&LISTEN, "l", ":12345", "The address to listen on")
	flag.StringVar(&CACHE_PATH, "c", "./cache", "The path to hath cache")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func main() {
	// 监听 TCP 端口
	ln, err := net.Listen("tcp", LISTEN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to listen on address")
	}
	defer ln.Close()
	log.Info().Msgf("Listening on %s", LISTEN)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error().Err(err).Msg("Failed to accept connection")
			continue
		}
		log.Info().Msgf("Accepted connection from %s", conn.RemoteAddr())
		go handleConn(conn)
	}
}
