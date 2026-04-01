package localfs

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"OpsPilot/internal/domain/knowledge"
)

// FileStore 将上传文件保存到本地文件系统。
type FileStore struct {
	rootDir string
}

// NewStore 创建本地文件存储。
func NewStore(rootDir string) *FileStore {
	return &FileStore{rootDir: strings.TrimSpace(rootDir)}
}

// Save 将上传文件写入目标目录，并返回落盘后的文件信息。
func (s *FileStore) Save(ctx context.Context, file *multipart.FileHeader) (*knowledge.StoredFile, error) {
	if file == nil {
		return nil, fmt.Errorf("请上传文件")
	}

	rootDir := s.rootDir
	if rootDir == "" {
		rootDir = "./docs/knowledge"
	}
	// 启动时自动补齐目录，避免首次上传因目录不存在而失败。
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %s: %w", rootDir, err)
	}

	fileName := filepath.Base(file.Filename)
	savePath := filepath.Join(rootDir, fileName)

	// 只使用基础文件名，避免通过上传文件名构造越级路径。
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		return nil, fmt.Errorf("创建保存文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	fileInfo, err := os.Stat(savePath)
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return &knowledge.StoredFile{
		FileName: fileName,
		FilePath: savePath,
		FileSize: fileInfo.Size(),
	}, nil
}
