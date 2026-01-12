package port

import (
	"context"

	"github.com/cloudwego/eino/components/model"
)

// ChatModelFactory 定义工作流层对 LLM ChatModel 的最小依赖（port）。
type ChatModelFactory interface {
	Get(ctx context.Context, name string) (model.BaseChatModel, error)
}
