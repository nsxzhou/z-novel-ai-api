package eino

import (
	"sync"

	einocallbacks "github.com/cloudwego/eino/callbacks"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"
)

var initOnce sync.Once

// Init 注册 Eino 全局 callbacks（进程级一次）。
func Init() {
	initOnce.Do(func() {
		handler := cbtemplate.NewHandlerHelper().
			ChatModel(newChatModelCallbackHandler()).
			Tool(newToolCallbackHandler()).
			Handler()
		einocallbacks.AppendGlobalHandlers(handler)
	})
}
