package milvus

import (
	"context"
	"encoding/json"

	retrievercomponent "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	cli "github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	denseTopK    = 5
	sparseTopK   = 5
	denseWeight  = 0.5
	sparseWeight = 0.5
)

type Retriever struct {
	store *Store
	topK  int
}

// NewRetriever 创建基于当前集合的向量检索器。
func (s *Store) NewRetriever(ctx context.Context, topK int) (retrievercomponent.Retriever, error) {
	if topK <= 0 {
		topK = 5
	}

	_ = s.client.LoadCollection(ctx, s.config.Collection, false)
	return &Retriever{
		store: s,
		topK:  topK,
	}, nil
}

func (r *Retriever) Retrieve(ctx context.Context, query string, opts ...retrievercomponent.Option) ([]*schema.Document, error) {
	embeddings, err := r.store.embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, err
	}

	sparseVector, err := buildSparseVector(query)
	if err != nil {
		return nil, err
	}

	denseSearchParam, err := entity.NewIndexAUTOINDEXSearchParam(1)
	if err != nil {
		return nil, err
	}
	sparseSearchParam, err := entity.NewIndexSparseInvertedSearchParam(0.2)
	if err != nil {
		return nil, err
	}

	requests := []*cli.ANNSearchRequest{
		cli.NewANNSearchRequest(
			DenseVectorField,
			entity.COSINE,
			"",
			[]entity.Vector{entity.FloatVector(toFloat32Vector(embeddings[0]))},
			denseSearchParam,
			denseTopK,
		),
		cli.NewANNSearchRequest(
			SparseVectorField,
			entity.IP,
			"",
			[]entity.Vector{sparseVector},
			sparseSearchParam,
			sparseTopK,
		),
	}

	results, err := r.store.client.HybridSearch(
		ctx,
		r.store.config.Collection,
		nil,
		r.topK,
		[]string{IDField, DocIDField, TitleField, ChunkIDField, ContentField, MetadataField},
		cli.NewWeightedReranker([]float64{denseWeight, sparseWeight}),
		requests,
	)
	if err != nil {
		return nil, err
	}

	documents := make([]*schema.Document, 0, len(results))
	seen := make(map[string]struct{})
	for _, result := range results {
		for idx := 0; idx < result.ResultCount; idx++ {
			id := columnString(result.IDs, idx)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}

			documents = append(documents, &schema.Document{
				ID:       id,
				Content:  columnString(result.Fields.GetColumn(ContentField), idx),
				MetaData: buildDocumentMetadata(result, idx),
			})
		}
	}

	return documents, nil
}

func buildDocumentMetadata(result cli.SearchResult, idx int) map[string]any {
	metadata := map[string]any{
		IDField:      columnString(result.IDs, idx),
		DocIDField:   columnString(result.Fields.GetColumn(DocIDField), idx),
		TitleField:   columnString(result.Fields.GetColumn(TitleField), idx),
		ChunkIDField: columnInt64(result.Fields.GetColumn(ChunkIDField), idx),
		"_score":     result.Scores[idx],
	}

	if raw := columnBytes(result.Fields.GetColumn(MetadataField), idx); len(raw) > 0 {
		var extra map[string]any
		if err := json.Unmarshal(raw, &extra); err == nil {
			for key, value := range extra {
				metadata[key] = value
			}
		}
	}

	return metadata
}

func columnString(column entity.Column, idx int) string {
	return column.(*entity.ColumnVarChar).Data()[idx]
}

func columnInt64(column entity.Column, idx int) int64 {
	return column.(*entity.ColumnInt64).Data()[idx]
}

func columnBytes(column entity.Column, idx int) []byte {
	return column.(*entity.ColumnJSONBytes).Data()[idx]
}
