package milvus

import (
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"github.com/yanyiwu/gojieba"
)

const sparseHashBucketSize = 1 << 20

var (
	jiebaOnce sync.Once
	jiebaCut  *gojieba.Jieba
)

func tokenize(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	jiebaOnce.Do(func() {
		jiebaCut = gojieba.NewJieba()
	})

	tokens := jiebaCut.CutForSearch(text, true)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" {
			continue
		}
		result = append(result, token)
	}
	return result
}

func buildSparseVector(text string) (entity.SparseEmbedding, error) {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return entity.NewSliceSparseEmbedding([]uint32{1}, []float32{1})
	}

	counts := make(map[uint32]float32, len(tokens))
	for _, token := range tokens {
		counts[tokenPosition(token)]++
	}

	positions := make([]uint32, 0, len(counts))
	for position := range counts {
		positions = append(positions, position)
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i] < positions[j]
	})

	values := make([]float32, 0, len(positions))
	total := float32(len(tokens))
	for _, position := range positions {
		values = append(values, counts[position]/total)
	}

	return entity.NewSliceSparseEmbedding(positions, values)
}

func tokenPosition(token string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(token))
	return hasher.Sum32()%sparseHashBucketSize + 1
}
