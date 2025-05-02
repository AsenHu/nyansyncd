package proto

func (f *FileInfo) Check() bool {
	// 检查宽度和高度 都应该介于 100 和 20000
	if f.Width < 100 || f.Width > 20000 {
		return false
	}
	if f.Height < 100 || f.Height > 20000 {
		return false
	}

	// 检查文件类型
	if f.Type > 3 {
		return false
	}

	/*
		检查文件大小
		 GIF 为 10 MiB
		JPG 为 20 MiB
		WEBP 为 20 MiB
		PNG 为 50 MiB
	*/
	switch f.Type {
	case GIF:
		if f.Size > 10*1024*1024 {
			return false
		}
	case JPG:
		if f.Size > 20*1024*1024 {
			return false
		}
	case WEBP:
		if f.Size > 20*1024*1024 {
			return false
		}
	case PNG:
		if f.Size > 50*1024*1024 {
			return false
		}
	}

	return true
}
