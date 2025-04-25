package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/AsenHu/nyansyncd/internal/finder"
	"github.com/AsenHu/nyansyncd/internal/header"
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

/*
handleControl 处理控制流

这里一共使用两个协程处理任务，一个协程专门接受数据，一个协程检查数据后发送数据
接受数据的协程收到数据后，使用 chan 发送数据到检查数据的协程，这里的 chan 是无缓冲的
以便让接受数据和处理数据的协程同步，但又不会让磁盘和网络 IO 互相阻塞
当 chan 关闭时，由检查数据的协程处理完数据后关闭连接
*/
func handleControl(conn net.Conn) {
	// 这里使用一个无缓冲的 chan 来传递 SHA1
	var sha1Chan chan [20]byte
	defer close(sha1Chan)

	// 开启一个协程来处理数据
	go func() {
		defer conn.Close()
		// 创建 SHA1 Finder
		var finder = finder.NewFinder(CACHE_PATH)

		for {
			// 从 chan 中读取 SHA1
			sha1, ok := <-sha1Chan
			if !ok {
				log.Info().Msg("SHA1 channel closed")
				return
			}

			// 检查 SHA1 是否存在
			exi, err := finder.Sha1Exists(sha1)
			// 忽略时间错误
			if err != nil && err != header.ErrInvalidTime {
				log.Error().Err(err).Msg("Failed to find file")
				conn.Close()
			}
			if !exi {
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

type dataChanMsg struct {
	info header.ImageHeader
	data []byte
}

/*
handleData 处理数据流

为了避免磁盘和网络 IO 互相阻塞，这里使用一个无缓冲的 chan 来传递数据
即 一个协程从 TCP 读取数据到内存，另一个协程从内存写入磁盘
当从 TCP 读取数据时，先读取 32 字节的头部数据，然后读取数据
读取数据时，应该计算数据的 SHA1，并且检查头部是否合法，只有合法的头部才会写入磁盘
如果头部不合法，则返回错误的 SHA1
*/
func handleData(conn net.Conn) {
	defer conn.Close()
	// 构建一个无缓冲的 chan 来传递数据
	var dataChan chan dataChanMsg
	defer close(dataChan)

	// 开启一个协程来把数据写入磁盘
	go func() {
		for {
			// 从 chan 中读取数据
			data, ok := <-dataChan
			if !ok {
				log.Info().Msg("Data channel closed")
				return
			}

			// 解析文件名
			fileName, err := data.info.ToString()
			if err != nil {
				if err == header.ErrInvalidTime {
					data.info.Time = time.Now()
				} else {
					log.Error().Err(err).Msg("Failed to parse file name")
					continue
				}
			}

			// 拼接文件名
			filePath := filepath.Join(CACHE_PATH, fmt.Sprintf("%02x", data.info.SHA1[0]), fmt.Sprintf("%02x", data.info.SHA1[1]), fileName)

			// 调用 os.MkdirAll 来创建目录
			if err := os.MkdirAll(filepath.Dir(filePath), 0644); err != nil {
				log.Error().Err(err).Msg("Failed to create directory")
				continue
			}

			// 写入文件
			if err := os.WriteFile(filePath, data.data, 0644); err != nil {
				log.Error().Err(err).Msg("Failed to write file")
				continue
			}
			log.Info().Msgf("File %s saved", filePath)
		}
	}()

	// 从连接读取数据
	for {
		// 读取头部数据
		var headerData [32]byte
		_, err := conn.Read(headerData[:])
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Connection closed by client")
				return
			}
			log.Error().Err(err).Msg("Failed to read header data")
			return
		}

		// 解析头部数据
		info, err := header.FromBytes(headerData)
		if err != nil {
			if err == header.ErrInvalidTime {
				log.Error().Err(err).Msg("Your computer's memory is unreliable; you should check your memory")
			}
			log.Error().Err(err).Msg("Failed to parse header data")
			return
		}

		// 创建一个哈希计算器
		hasher := sha1.New()

		// 使用 TeeReader 读取数据并计算 SHA1
		data := make([]byte, info.Size)
		tee := io.TeeReader(conn, hasher)
		_, err = tee.Read(data)
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("Connection closed by client")
				return
			}
			log.Error().Err(err).Msg("Failed to read data")
			return
		}

		// 验证计算的 SHA1 是否与头部中的 SHA1 匹配
		calculatedSHA1 := ([20]byte)(hasher.Sum(nil))
		if calculatedSHA1 != info.SHA1 {
			header, err := info.ToString()
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse header data whren SHA1 mismatch")
			}
			log.Error().
				Str("Header", header).
				Str("Calculated", fmt.Sprintf("%x", calculatedSHA1)).
				Msg("SHA1 mismatch")

			// 返回错误的 SHA1
			/*
				sender 暂时无法处理这些数据
				go func() {
					if _, err := conn.Write(info.SHA1[:]); err != nil {
						log.Error().Err(err).Msg("Failed to write SHA1")
						conn.Close()
					}
				}()
			*/

			continue
		}

		dataChan <- dataChanMsg{info: info, data: data}
	}
}
