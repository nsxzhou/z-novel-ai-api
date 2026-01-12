package chain

import (
	"context"
	"fmt"
	"strings"
	"sync"

	openaiopts "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	llmctx "z-novel-ai-api/internal/domain/service"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	wfnode "z-novel-ai-api/internal/workflow/node"
	workflowport "z-novel-ai-api/internal/workflow/port"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
	"z-novel-ai-api/pkg/logger"
)

type ProjectCreationChain struct {
	factory workflowport.ChatModelFactory

	chainOnce sync.Once
	chain     compose.Runnable[*wfmodel.ProjectCreationGenerateInput, *schema.Message]
	chainErr  error
}

func NewProjectCreationChain(factory workflowport.ChatModelFactory) *ProjectCreationChain {
	return &ProjectCreationChain{factory: factory}
}

func (c *ProjectCreationChain) Invoke(ctx context.Context, in *wfmodel.ProjectCreationGenerateInput) (*schema.Message, error) {
	if c == nil || c.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	chain, err := c.getChain()
	if err != nil {
		return nil, err
	}
	return chain.Invoke(ctx, in)
}

type projectCreationChainState struct {
	In       *wfmodel.ProjectCreationGenerateInput
	Messages []*schema.Message
	OutMsg   *schema.Message
}

func (c *ProjectCreationChain) getChain() (compose.Runnable[*wfmodel.ProjectCreationGenerateInput, *schema.Message], error) {
	c.chainOnce.Do(func() {
		c.chain, c.chainErr = c.buildChain(context.Background())
	})
	return c.chain, c.chainErr
}

func (c *ProjectCreationChain) buildChain(ctx context.Context) (compose.Runnable[*wfmodel.ProjectCreationGenerateInput, *schema.Message], error) {
	chain := compose.NewChain[*wfmodel.ProjectCreationGenerateInput, *schema.Message]()

	chain.AppendLambda(
		compose.InvokableLambda(func(_ context.Context, in *wfmodel.ProjectCreationGenerateInput) (*projectCreationChainState, error) {
			if in == nil {
				return nil, fmt.Errorf("input is nil")
			}
			return &projectCreationChainState{In: in}, nil
		}),
		compose.WithNodeName("project_creation.init"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *projectCreationChainState) (*projectCreationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			msgs, err := formatProjectCreationMessages(ctx, st.In)
			if err != nil {
				return nil, err
			}
			st.Messages = msgs
			return st, nil
		}),
		compose.WithNodeName("project_creation.template"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(ctx context.Context, st *projectCreationChainState) (*projectCreationChainState, error) {
			if st == nil || st.In == nil {
				return nil, fmt.Errorf("state is nil")
			}
			if c.factory == nil {
				return nil, fmt.Errorf("llm factory not configured")
			}

			ctx = llmctx.WithWorkflowProvider(ctx, "project_creation_generate", strings.TrimSpace(st.In.Provider))
			chatModel, err := c.factory.Get(ctx, strings.TrimSpace(st.In.Provider))
			if err != nil {
				return nil, err
			}

			outMsg, err := chatModel.Generate(ctx, st.Messages, buildProjectCreationModelOptions(st.In, true)...)
			if err != nil && wfnode.IsResponseFormatUnsupportedError(err) {
				logger.Warn(ctx, "llm json_schema not supported, fallback to prompt-only",
					"provider", strings.TrimSpace(st.In.Provider),
					"model", strings.TrimSpace(st.In.Model),
					"error", err.Error(),
				)
				outMsg, err = chatModel.Generate(ctx, st.Messages, buildProjectCreationModelOptions(st.In, false)...)
			}
			if err != nil {
				return nil, err
			}
			if outMsg == nil {
				return nil, fmt.Errorf("empty llm response")
			}
			st.OutMsg = outMsg
			return st, nil
		}),
		compose.WithNodeName("project_creation.llm"),
	)

	chain.AppendLambda(
		compose.InvokableLambda(func(_ context.Context, st *projectCreationChainState) (*schema.Message, error) {
			if st == nil || st.OutMsg == nil {
				return nil, fmt.Errorf("state is nil")
			}
			return st.OutMsg, nil
		}),
		compose.WithNodeName("project_creation.finalize"),
	)

	return chain.Compile(ctx)
}

func formatProjectCreationMessages(ctx context.Context, in *wfmodel.ProjectCreationGenerateInput) ([]*schema.Message, error) {
	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptProjectCreationV1)
	if err != nil {
		return nil, err
	}
	draft := "{}"
	if len(in.Draft) > 0 {
		draft = strings.TrimSpace(string(in.Draft))
		if draft == "" {
			draft = "{}"
		}
	}
	vars := map[string]any{
		"stage":             strings.TrimSpace(in.Stage),
		"draft_json":        draft,
		"prompt":            strings.TrimSpace(in.Prompt),
		"attachments_block": wfnode.BuildAttachmentsBlock(in.Attachments),
	}
	return tpl.Format(ctx, vars)
}

func buildProjectCreationModelOptions(in *wfmodel.ProjectCreationGenerateInput, enableSchema bool) []model.Option {
	opts := make([]model.Option, 0, 4)
	if in == nil {
		return opts
	}
	if in.Temperature != nil {
		opts = append(opts, model.WithTemperature(*in.Temperature))
	}
	if in.MaxTokens != nil {
		opts = append(opts, model.WithMaxTokens(*in.MaxTokens))
	}
	if strings.TrimSpace(in.Model) != "" {
		opts = append(opts, model.WithModel(strings.TrimSpace(in.Model)))
	}

	if enableSchema {
		opts = append(opts, openaiopts.WithExtraFields(map[string]any{
			"response_format": map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"name":   "project_creation",
					"strict": false,
					"schema": projectCreationJSONSchema(),
				},
			},
		}))
	}

	return opts
}

func projectCreationJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"assistant_message", "stage", "draft", "action", "requires_confirmation"},
		"properties": map[string]any{
			"assistant_message": map[string]any{"type": "string"},
			"stage": map[string]any{
				"type": "string",
				"enum": []any{"discover", "narrow", "draft", "confirm"},
			},
			"draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"title", "description", "genre"},
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"genre":       map[string]any{"type": "string"},
				},
			},
			"action": map[string]any{
				"type": "string",
				"enum": []any{"none", "propose_creation", "create_project"},
			},
			"requires_confirmation": map[string]any{"type": "boolean"},
			"project": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"title", "description"},
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"genre":       map[string]any{"type": "string"},
				},
			},
		},
	}
}
