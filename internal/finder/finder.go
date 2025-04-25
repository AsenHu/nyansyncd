package finder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AsenHu/nyansyncd/internal/header"
)

/*
这个包提供一个文件查找器的实现
也就是 sha1Exists

这个查找器核心思想是，利用 os.readDir 会按照文件名的字典序来读取文件
这样，当查找的 HASH 值是有序的，并且是在给定目录下的
那么我们就可以利用这个特性来加速查找

查找器假设 abcdef 文件，应该放在给定目录的 ./ab/cd/abcdef 的位置
*/

type Finder struct {
	path          string
	files         []os.DirEntry
	cursor        uint64
	lastCheckHash [20]byte
}

func NewFinder(path string) *Finder {
	return &Finder{
		path:          path,
		cursor:        0,
		lastCheckHash: [20]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
	}
}

/*
这个函数检查前，需要先知道是否要打开一个新的目录
要打开的目标目录是 hash 的前两位，因此只要比较 lastCheckHash 和 hash 的前两位是否相同即可
但这有一个特殊的情况，即 lastCheckHash 和 hash 的前两位相同，但 lastCheckHash 比较大
这意味着对方重新列出了目录，所以出现了乱序的情况，这种情况也需要重新打开目录

对于无需重新打开目录，或已经打开目录的情况，按这个方法操作
1. 读取位于 cursor 的文件，提取文件名中 hash
2. 比较文件名的 hash 和传入的 hash
这时会出现三种情况
 1. 相同，返回 true
 2. 文件名的 hash 小于传入的 hash，继续读取下一个文件（cursor++ 后再次比较）
 3. 文件名的 hash 大于传入的 hash，返回 false

3. 当比较完成后，把 lastCheckHash 设置为 hash，表示该 hash 之前的文件都已经检查过了

注意，当 cursor 超过了文件列表的长度时，表示已经检查完了所有的文件，直接返回 false，但不要关闭文件列表
这时可能仍然会需要检查更多的无需重新打开目录的文件，此后一律返回 false，并仅更新 lastCheckHash
*/
func (f *Finder) Sha1Exists(hash [20]byte) (bool, error) {
	// 先检查是否需要打开新的目录(只有前两位相同并且 hash 大于等于 lastCheckHash 的情况可以不打开目录)
	if f.lastCheckHash[0] != hash[0] || f.lastCheckHash[1] != hash[1] || byteCompare(f.lastCheckHash, hash) == 1 {
		// 需要打开新的目录
		if err := f.openDir([2]byte{hash[0], hash[1]}); err != nil {
			return false, err
		}
	}

	for {
		// 检查是否已经检查完了所有的文件
		if f.cursor >= uint64(len(f.files)) {
			// 已经检查完了所有的文件
			f.lastCheckHash = hash
			return false, nil
		}

		// 读取文件名
		fileInfo, err := header.FromString(f.files[f.cursor].Name())
		if err != nil {
			return false, err
		}

		// 比较文件名的 hash 和传入的 hash
		switch byteCompare(fileInfo.SHA1, hash) {
		case -1:
			// 文件名的 hash 小于传入的 hash，继续读取下一个文件
			f.cursor++
			continue
		case 0:
			// 相同，返回 true
			f.cursor++
			f.lastCheckHash = hash
			return true, nil
		case 1:
			// 文件名的 hash 大于传入的 hash，返回 false
			f.lastCheckHash = hash
			return false, nil
		}
	}
}

func (f *Finder) openDir(pathByte [2]byte) error {
	// 把 pathByte 转换为字符串
	path := filepath.Join(f.path, fmt.Sprintf("%02x", pathByte[0]), fmt.Sprintf("%02x", pathByte[1]))

	// 确保这个目录存在
	if err := os.MkdirAll(path, 0644); err != nil {
		return err
	}

	// 读取目录
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	f.files = files
	f.cursor = 0
	f.lastCheckHash = [20]byte{pathByte[0], pathByte[1]}
	return nil
}

func byteCompare(a, b [20]byte) int8 {
	for i := range 20 {
		if a[i] < b[i] {
			return -1
		} else if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
