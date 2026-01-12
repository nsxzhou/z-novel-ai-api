package service

import (
	"context"
	"strings"
)

type llmCtxKey string

const (
	llmCtxKeyWorkflow llmCtxKey = "llm_workflow"
	llmCtxKeyProvider llmCtxKey = "llm_provider"
)

func WithWorkflow(ctx context.Context, workflow string) context.Context {
	if ctx == nil {
		return nil
	}
	w := strings.TrimSpace(workflow)
	if w == "" {
		return ctx
	}
	return context.WithValue(ctx, llmCtxKeyWorkflow, w)
}

func WithProvider(ctx context.Context, provider string) context.Context {
	if ctx == nil {
		return nil
	}
	p := strings.TrimSpace(provider)
	if p == "" {
		return ctx
	}
	return context.WithValue(ctx, llmCtxKeyProvider, p)
}

func WithWorkflowProvider(ctx context.Context, workflow, provider string) context.Context {
	return WithProvider(WithWorkflow(ctx, workflow), provider)
}

func WorkflowFromContext(ctx context.Context) string {
	if ctx == nil {
		return "unknown"
	}
	v := ctx.Value(llmCtxKeyWorkflow)
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return strings.TrimSpace(s)
}

func ProviderFromContext(ctx context.Context) string {
	if ctx == nil {
		return "unknown"
	}
	v := ctx.Value(llmCtxKeyProvider)
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return strings.TrimSpace(s)
}
