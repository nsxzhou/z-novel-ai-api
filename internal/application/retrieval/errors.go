package retrieval

import "errors"

var (
	// ErrVectorDisabled 表示向量检索/索引能力未配置（Milvus 或 Embedder 不可用）。
	ErrVectorDisabled = errors.New("vector retrieval is disabled")
)
