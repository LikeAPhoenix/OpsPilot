package milvus

import "github.com/milvus-io/milvus-sdk-go/v2/entity"

// Milvus 集合字段名称常量。
const (
	IDField           = "id"
	DocIDField        = "doc_id"
	TitleField        = "title"
	ChunkIDField      = "chunk_id"
	DenseVectorField  = "dense_vector"
	SparseVectorField = "sparse_vector"
	ContentField      = "content"
	MetadataField     = "metadata"
)

const (
	defaultDenseVectorDim = 2048
	maxVarcharLength      = "8192"
)

// Fields 返回知识库集合的字段定义。
func Fields() []*entity.Field {
	return []*entity.Field{
		{
			Name:     IDField,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": "256",
			},
			PrimaryKey: true,
		},
		{
			Name:     DocIDField,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": "256",
			},
		},
		{
			Name:     TitleField,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": "1024",
			},
		},
		{
			Name:     ChunkIDField,
			DataType: entity.FieldTypeInt64,
		},
		{
			Name:     DenseVectorField,
			DataType: entity.FieldTypeFloatVector,
			TypeParams: map[string]string{
				"dim": "2048",
			},
		},
		{
			Name:     SparseVectorField,
			DataType: entity.FieldTypeSparseVector,
		},
		{
			Name:     ContentField,
			DataType: entity.FieldTypeVarChar,
			TypeParams: map[string]string{
				"max_length": maxVarcharLength,
			},
		},
		{
			Name:     MetadataField,
			DataType: entity.FieldTypeJSON,
		},
	}
}
