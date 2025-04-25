package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AsenHu/nyansyncd/internal/dialer"
	"github.com/AsenHu/nyansyncd/internal/header"
	"github.com/rs/zerolog/log"

	"github.com/rs/zerolog"
)

/*
这个文件是 sender 的主程序
整个协议的具体工作流程见 reveiver.go 的开头
这里讲述一下 sender 的工作方式

这里有两个协程，其中一个协程负责将本地已有的文件 hash，发送给 receiver
另一个协程负责将 receiver 返回的 hash，从本地翻出来，发送出去

第一个协程首先会通过 lister.go 获取一组文件的列表，并逐个发送给 receiver
然后把创建 TCP 得到的 Reader 传递给第二个协程
如果对方主动关闭连接，则抛弃当前从 lister.go 获取的文件列表，获取下一组新的文件列表
如果第二个协程请求关闭连接，则主动关闭连接并创建新的连接和协程，继续发送

第二个协程应该由第一个协程来唤醒，唤醒后会得到一个控制流的 Reader 和 close 连接的方法
从 Reader 中读取 SHA1，并从本地获取文件，然后从自己创建的数据流发出去。
但有以下几种可能的错误
1. 任何情况的永远无法从 Reader 读取数据，这种情况下，说明第二个协程没有可能处理更多任务了，在完成了已有的任务后，请求关闭控制流和数据流的连接
2. 对方从控制流发回了不存在的 SHA1，这说明控制流出现了故障，因此请求关闭控制流，但这意味着自己的工作也即将结束，因此在完成已有的任务后，关闭数据流的连接

对于第二个协程自己创建的数据流，在正常情况下，是从控制流的 Reader 和数据流的 Reader 里，获取文件 hash 并发送的
但有以下几种可能的错误
1. 对方关闭了数据流连接，这说明对方不希望继续接收这个 hash 分片的文件了，这时应该关闭控制流和数据流的连接
2. 对方从数据流返回了错误的 SHA1，这说明 TCP 的连接出现了错误，这时应该关闭数据流的连接，并重建新的数据流
*/

var (
	DIAL       string // 监听的地址
	CACHE_PATH string // 缓存的路径
)

func init() {
	flag.StringVar(&DIAL, "d", "", "The address to dial")
	flag.StringVar(&CACHE_PATH, "c", "./cache", "The path to hath cache")
	flag.Parse()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if DIAL == "" {
		log.Fatal().Msg("Please specify the address to dial")
	}
}

func main() {
	// 创建一个 dialer
	dialer := dialer.NewDialer(DIAL)

	// 第二个协程
	go func() {
		// 它的工作就是循环从 ctrlr 读取 20 字节，然后把图片从 dataw 发出去
		for {
			// 读取 SHA1
			var sha1 [20]byte
			dialer.CtrlR(sha1[:])

			// 获取文件名
			/*
				1. 根据 SHA1 前两个字节，获取目录
				2. 遍历目录，查看寻找文件开头是目标 SHA1
			*/
			dir := filepath.Join(CACHE_PATH, fmt.Sprintf("%02x", sha1[0]), fmt.Sprintf("%02x", sha1[1]))
			files, err := os.ReadDir(dir)
			if err != nil {
				log.Error().Err(err).Msg("Failed to read directory")
				continue
			}
			for _, file := range files {
				fileInfo, _ := header.FromString(file.Name())
				if fileInfo.SHA1 == sha1 {
					// 获取文件 Mtime
					fileStatInfo, err := os.Stat(filepath.Join(dir, file.Name()))
					if err != nil {
						log.Error().Err(err).Msg("Failed to get file stat")
						continue
					}
					fileInfo.Time = fileStatInfo.ModTime()
					// 发送文件头
					headerBytes, err := fileInfo.ToBytes()
					if err != nil {
						log.Error().Err(err).Msg("Failed to convert file info to bytes")
						continue
					}
					dialer.DataW(headerBytes[:])
					// 发送文件数据
					buffer, err := os.ReadFile(filepath.Join(dir, file.Name()))
					if err != nil {
						log.Error().Err(err).Msg("Failed to read file")
						continue
					}
					dialer.DataW(buffer)
					break
				}
			}
		}
	}()

	// 第一个协程
	// 它的工作就是循环从 lister 读取 20 字节，然后把图片从 ctrlw 发出去
	for {
		// 列出 cache 目录
		rootList, err := os.ReadDir(CACHE_PATH)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read directory")
			continue
		}
		for _, dir1 := range rootList {
			// 读取目录
			dir1List, err := os.ReadDir(filepath.Join(CACHE_PATH, dir1.Name()))
			if err != nil {
				log.Error().Err(err).Msg("Failed to read directory")
				continue
			}
			for _, dir2 := range dir1List {
				// 读取目录
				dir2List, err := os.ReadDir(filepath.Join(CACHE_PATH, dir1.Name(), dir2.Name()))
				if err != nil {
					log.Error().Err(err).Msg("Failed to read directory")
					continue
				}
				for _, file := range dir2List {
					// 读取文件名
					fileInfo, err := header.FromString(file.Name())
					if err != nil {
						log.Error().Err(err).Msg("Failed to parse file name")
						continue
					}
					// 发送 SHA1
					dialer.CtrlW(fileInfo.SHA1[:])
				}
			}
		}
	}
}
