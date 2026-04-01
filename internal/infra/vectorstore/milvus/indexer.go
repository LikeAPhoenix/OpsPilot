package milvus

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

type Indexer struct {
	store *Store
}

// NewIndexer 创建基于当前集合的 Milvus 索引器。
func (s *Store) NewIndexer(ctx context.Context) (indexer.Indexer, error) {
	return &Indexer{store: s}, nil
}

func (i *Indexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	texts := make([]string, 0, len(docs))
	for _, doc := range docs {
		texts = append(texts, doc.Content)
	}

	embeddings, err := i.store.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(docs))
	docIDs := make([]string, 0, len(docs))
	titles := make([]string, 0, len(docs))
	chunkIDs := make([]int64, 0, len(docs))
	denseVectors := make([][]float32, 0, len(docs))
	sparseVectors := make([]entity.SparseEmbedding, 0, len(docs))
	contents := make([]string, 0, len(docs))
	metadatas := make([][]byte, 0, len(docs))

	for idx, doc := range docs {
		source := metadataString(doc.MetaData, "_source")
		docID := sourceDocID(source)
		chunkID := int64(idx)
		title := metadataString(doc.MetaData, "title")
		if title == "" {
			title = filepath.Base(source)
		}

		sparseVector, err := buildSparseVector(doc.Content)
		if err != nil {
			return nil, err
		}

		meta := cloneMetadata(doc.MetaData)
		meta[DocIDField] = docID
		meta[TitleField] = title
		meta[ChunkIDField] = chunkID

		encodedMeta, err := json.Marshal(meta)
		if err != nil {
			return nil, err
		}

		ids = append(ids, buildPrimaryID(docID, chunkID))
		docIDs = append(docIDs, docID)
		titles = append(titles, title)
		chunkIDs = append(chunkIDs, chunkID)
		denseVectors = append(denseVectors, toFloat32Vector(embeddings[idx]))
		sparseVectors = append(sparseVectors, sparseVector)
		contents = append(contents, doc.Content)
		metadatas = append(metadatas, encodedMeta)
	}

	_, err = i.store.client.Insert(
		ctx,
		i.store.config.Collection,
		"",
		entity.NewColumnVarChar(IDField, ids),
		entity.NewColumnVarChar(DocIDField, docIDs),
		entity.NewColumnVarChar(TitleField, titles),
		entity.NewColumnInt64(ChunkIDField, chunkIDs),
		entity.NewColumnFloatVector(DenseVectorField, defaultDenseVectorDim, denseVectors),
		entity.NewColumnSparseVectors(SparseVectorField, sparseVectors),
		entity.NewColumnVarChar(ContentField, contents),
		entity.NewColumnJSONBytes(MetadataField, metadatas),
	)
	if err != nil {
		return nil, err
	}

	if err := i.store.client.Flush(ctx, i.store.config.Collection, false); err != nil {
		return nil, err
	}

	return ids, nil
}

func buildPrimaryID(docID string, chunkID int64) string {
	return docID + ":" + int64String(chunkID)
}

func cloneMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func metadataString(input map[string]any, key string) string {
	if len(input) == 0 {
		return ""
	}
	value, ok := input[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func toFloat32Vector(values []float64) []float32 {
	vector := make([]float32, 0, len(values))
	for _, value := range values {
		vector = append(vector, float32(value))
	}
	return vector
}

func int64String(value int64) string {
	return strconv.FormatInt(value, 10)
}
