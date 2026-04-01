package knowledge

import (
	"context"
	"mime/multipart"

	domainknowledge "OpsPilot/internal/domain/knowledge"
)

// FileStore 定义上传文件的落盘能力。
type FileStore interface {
	Save(ctx context.Context, file *multipart.FileHeader) (*domainknowledge.StoredFile, error)
}

// Indexer 定义知识索引刷新能力。
type Indexer interface {
	Refresh(ctx context.Context, path string) error
}

// Service 负责协调文件存储与知识索引更新。
type Service struct {
	fileStore FileStore
	indexer   Indexer
}

// NewService 创建知识库应用服务。
func NewService(fileStore FileStore, indexer Indexer) *Service {
	return &Service{
		fileStore: fileStore,
		indexer:   indexer,
	}
}

// IndexPath 根据给定路径刷新知识索引。
func (s *Service) IndexPath(ctx context.Context, path string) error {
	return s.indexer.Refresh(ctx, path)
}

// UploadFile 先保存上传文件，再立即触发索引刷新，保证新文档可被检索到。
func (s *Service) UploadFile(ctx context.Context, file *multipart.FileHeader) (*domainknowledge.StoredFile, error) {
	storedFile, err := s.fileStore.Save(ctx, file)
	if err != nil {
		return nil, err
	}
	if err := s.indexer.Refresh(ctx, storedFile.FilePath); err != nil {
		return nil, err
	}
	return storedFile, nil
}
