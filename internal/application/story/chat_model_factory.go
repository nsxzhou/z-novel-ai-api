package story

import (
	"context"

	"github.com/cloudwego/eino/components/model"
)

// ChatModelFactory 定义应用层对 LLM ChatModel 的最小依赖（port）。
// 由基础设施层提供具体实现（例如 EinoFactory）。
type ChatModelFactory interface {
	Get(ctx context.Context, name string) (model.BaseChatModel, error)
}
