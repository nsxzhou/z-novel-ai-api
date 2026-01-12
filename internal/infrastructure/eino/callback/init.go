package callback

import (
	"sync"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"

	"z-novel-ai-api/internal/domain/service"
)

var initOnce sync.Once

// Init 注册 Eino 全局 callbacks（进程级一次）。
func Init(usageRecorder service.LLMUsageRecorder, tenantIDGetter TenantIDGetter) {
	initOnce.Do(func() {
		handler := cbtemplate.NewHandlerHelper().
			ChatModel(newChatModelCallbackHandler(usageRecorder, tenantIDGetter)).
			Tool(newToolCallbackHandler()).
			Handler()
		einocallbacks.AppendGlobalHandlers(handler)
	})
}
