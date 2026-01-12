package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openaiopts "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"

	llmctx "z-novel-ai-api/internal/domain/service"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	wfnode "z-novel-ai-api/internal/workflow/node"
	workflowprompt "z-novel-ai-api/internal/workflow/prompt"
)

func (g *ArtifactPipeline) ScanConflicts(ctx context.Context, in *wfmodel.ArtifactConflictScanInput) (*wfmodel.ArtifactConflictScanOutput, error) {
	if g == nil || g.factory == nil {
		return nil, fmt.Errorf("llm factory not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	if strings.TrimSpace(string(in.NewArtifact)) == "" {
		return nil, fmt.Errorf("new artifact json is empty")
	}

	tpl, err := defaultPromptRegistry.ChatTemplate(workflowprompt.PromptArtifactConflictScanV1)
	if err != nil {
		return nil, err
	}

	projectBrief, _ := json.Marshal(map[string]any{
		"title":       strings.TrimSpace(in.ProjectTitle),
		"description": strings.TrimSpace(in.ProjectDescription),
		"genre":       strings.TrimSpace(in.ProjectGenre),
	})

	formatJSON := func(b json.RawMessage, maxRunes int) string {
		s := strings.TrimSpace(string(b))
		if s == "" {
			return "null"
		}
		return wfnode.TruncateByRunes(s, maxRunes)
	}

	vars := map[string]any{
		"project_title":           strings.TrimSpace(in.ProjectTitle),
		"project_description":     strings.TrimSpace(in.ProjectDescription),
		"artifact_type":           strings.TrimSpace(string(in.Type)),
		"project_brief_json":      wfnode.TruncateByRunes(strings.TrimSpace(string(projectBrief)), 4000),
		"current_worldview_json":  formatJSON(in.CurrentWorldview, 20000),
		"current_characters_json": formatJSON(in.CurrentCharacters, 20000),
		"current_outline_json":    formatJSON(in.CurrentOutline, 20000),
		"current_artifact_json":   formatJSON(in.CurrentArtifact, 20000),
		"new_artifact_json":       formatJSON(in.NewArtifact, 40000),
	}

	msgs, err := tpl.Format(ctx, vars)
	if err != nil {
		return nil, err
	}

	ctx = llmctx.WithWorkflowProvider(ctx, "artifact_conflict_scan", strings.TrimSpace(in.Provider))
	chatModel, err := g.factory.Get(ctx, strings.TrimSpace(in.Provider))
	if err != nil {
		return nil, err
	}

	outMsg, err := chatModel.Generate(ctx, msgs, buildArtifactConflictScanModelOptions(in, true)...)
	if err != nil && wfnode.IsResponseFormatUnsupportedError(err) {
		outMsg, err = chatModel.Generate(ctx, msgs, buildArtifactConflictScanModelOptions(in, false)...)
	}
	if err != nil {
		return nil, err
	}
	if outMsg == nil {
		return nil, fmt.Errorf("empty llm response")
	}

	raw := wfnode.ExtractJSONObject(outMsg.Content)
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty conflict scan output")
	}

	var parsed struct {
		Conflicts []wfmodel.ArtifactConflict `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse conflict scan json: %w", err)
	}

	conflicts := normalizeConflicts(parsed.Conflicts)
	meta := wfmodel.LLMUsageMeta{
		Provider:    strings.TrimSpace(in.Provider),
		Model:       strings.TrimSpace(in.Model),
		GeneratedAt: time.Now().UTC(),
	}
	if in.Temperature != nil {
		meta.Temperature = float64(*in.Temperature)
	}
	if outMsg.ResponseMeta != nil && outMsg.ResponseMeta.Usage != nil {
		meta.PromptTokens = outMsg.ResponseMeta.Usage.PromptTokens
		meta.CompletionTokens = outMsg.ResponseMeta.Usage.CompletionTokens
	}

	return &wfmodel.ArtifactConflictScanOutput{
		Conflicts: conflicts,
		Raw:       raw,
		Meta:      meta,
	}, nil
}

func buildArtifactConflictScanModelOptions(in *wfmodel.ArtifactConflictScanInput, enableSchema bool) []model.Option {
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
					"name":   "artifact_conflict_scan",
					"strict": false,
					"schema": artifactConflictScanJSONSchema(),
				},
			},
		}))
	}
	return opts
}

func artifactConflictScanJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"conflicts"},
		"properties": map[string]any{
			"conflicts": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"severity", "message"},
					"properties": map[string]any{
						"severity": map[string]any{
							"type": "string",
							"enum": []any{string(wfmodel.ArtifactConflictSeverityHigh), string(wfmodel.ArtifactConflictSeverityMedium), string(wfmodel.ArtifactConflictSeverityLow)},
						},
						"message":      map[string]any{"type": "string"},
						"existing_ref": map[string]any{"type": "string"},
						"new_ref":      map[string]any{"type": "string"},
						"suggestion":   map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func normalizeConflicts(in []wfmodel.ArtifactConflict) []wfmodel.ArtifactConflict {
	if len(in) == 0 {
		return nil
	}
	out := make([]wfmodel.ArtifactConflict, 0, len(in))
	for i := range in {
		c := in[i]
		c.Message = strings.TrimSpace(c.Message)
		if c.Message == "" {
			continue
		}
		if c.Severity != wfmodel.ArtifactConflictSeverityHigh &&
			c.Severity != wfmodel.ArtifactConflictSeverityMedium &&
			c.Severity != wfmodel.ArtifactConflictSeverityLow {
			c.Severity = wfmodel.ArtifactConflictSeverityLow
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
