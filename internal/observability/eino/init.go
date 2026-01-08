package eino

import (
	"sync"
	"z-novel-ai-api/internal/domain/repository"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
)

var initOnce sync.Once

// Init 注册 Eino 全局 callbacks（进程级一次）。
func Init(tenantRepo repository.TenantRepository, llmRepo repository.LLMUsageEventRepository, tenantCtxMgr repository.TenantContextManager) {
	initOnce.Do(func() {
		handler := cbtemplate.NewHandlerHelper().
			ChatModel(newChatModelCallbackHandler(tenantRepo, llmRepo, tenantCtxMgr)).
			Tool(newToolCallbackHandler()).
			Handler()
		einocallbacks.AppendGlobalHandlers(handler)
	})
}
