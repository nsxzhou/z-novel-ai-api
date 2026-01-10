package story

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	appretrieval "z-novel-ai-api/internal/application/retrieval"
	"z-novel-ai-api/internal/domain/entity"
)

const (
	toolNameArtifactGetActive = "artifact_get_active"
	toolNameArtifactSearch    = "artifact_search"
	toolNameProjectGetBrief   = "project_get_brief"
)

type artifactGetActiveTool struct {
	in *ArtifactGenerateInput
}

func newArtifactGetActiveTool(in *ArtifactGenerateInput) *artifactGetActiveTool {
	return &artifactGetActiveTool{in: in}
}

func (t *artifactGetActiveTool) GetType() string { return toolNameArtifactGetActive }

func (t *artifactGetActiveTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: toolNameArtifactGetActive,
		Desc: "读取指定类型的当前设定 JSON（世界观/角色/大纲/小说基底）。用于在需要时按需加载上下文。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"type": {
				Type:     schema.String,
				Desc:     "构件类型：novel_foundation/worldview/characters/outline",
				Required: true,
				Enum:     []string{string(entity.ArtifactTypeNovelFoundation), string(entity.ArtifactTypeWorldview), string(entity.ArtifactTypeCharacters), string(entity.ArtifactTypeOutline)},
			},
		}),
	}, nil
}

func (t *artifactGetActiveTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)

	reqType := entity.ArtifactType(strings.TrimSpace(args.Type))
	if reqType == "" {
		return "", fmt.Errorf("missing type")
	}

	var content json.RawMessage
	if t.in != nil {
		switch reqType {
		case entity.ArtifactTypeWorldview:
			content = t.in.CurrentWorldview
		case entity.ArtifactTypeCharacters:
			content = t.in.CurrentCharacters
		case entity.ArtifactTypeOutline:
			content = t.in.CurrentOutline
		case entity.ArtifactTypeNovelFoundation:
			if t.in.Type == entity.ArtifactTypeNovelFoundation {
				content = t.in.CurrentArtifactRaw
			}
		default:
			if t.in.Type == reqType {
				content = t.in.CurrentArtifactRaw
			}
		}
	}

	out := struct {
		Type   string          `json:"type"`
		Exists bool            `json:"exists"`
		JSON   json.RawMessage `json:"json"`
	}{
		Type:   string(reqType),
		Exists: len(content) > 0,
		JSON:   content,
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

type artifactSearchTool struct {
	engine *appretrieval.Engine
	in     *ArtifactGenerateInput
}

func newArtifactSearchTool(engine *appretrieval.Engine, in *ArtifactGenerateInput) *artifactSearchTool {
	return &artifactSearchTool{engine: engine, in: in}
}

func (t *artifactSearchTool) GetType() string { return toolNameArtifactSearch }

func (t *artifactSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: toolNameArtifactSearch,
		Desc: "对项目的设定索引做语义检索（优先向量召回，降级为字符串匹配），返回命中片段用于定位 key/name/设定条目。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "检索关键词", Required: true},
			"type":  {Type: schema.String, Desc: "可选：限定构件类型（novel_foundation/worldview/characters/outline）"},
			"top_k": {Type: schema.Integer, Desc: "可选：返回命中条数，默认 5"},
		}),
	}, nil
}

func (t *artifactSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query string `json:"query"`
		Type  string `json:"type,omitempty"`
		TopK  int    `json:"top_k,omitempty"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		b, _ := json.Marshal(map[string]any{"error": fmt.Sprintf("invalid arguments: %v", err)})
		return string(b), nil
	}
	q := strings.TrimSpace(args.Query)
	if q == "" {
		b, _ := json.Marshal(map[string]any{"error": "query is required"})
		return string(b), nil
	}
	topK := args.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	type hit struct {
		ArtifactType string  `json:"artifact_type,omitempty"`
		RefPath      string  `json:"ref_path,omitempty"`
		Snippet      string  `json:"snippet"`
		Score        float64 `json:"score,omitempty"`
		Source       string  `json:"source,omitempty"`
	}
	var hits []hit

	// 1) 向量召回（可降级）
	disabledReason := ""
	segmentTypes := appretrieval.AllArtifactSegmentTypes()
	filterType := entity.ArtifactType(strings.TrimSpace(args.Type))
	if filterType != "" {
		segType := appretrieval.ArtifactSegmentType(filterType)
		if segType == "" {
			b, _ := json.Marshal(map[string]any{"error": fmt.Sprintf("invalid type: %s", strings.TrimSpace(args.Type))})
			return string(b), nil
		}
		segmentTypes = []string{segType}
	}

	if t != nil && t.engine != nil && t.in != nil {
		out, err := t.engine.Search(ctx, appretrieval.SearchInput{
			TenantID:        strings.TrimSpace(t.in.TenantID),
			ProjectID:       strings.TrimSpace(t.in.ProjectID),
			Query:           q,
			TopK:            topK,
			SegmentTypes:    segmentTypes,
			IncludeEntities: false,
		})
		if err != nil {
			disabledReason = err.Error()
		} else if out != nil {
			disabledReason = strings.TrimSpace(out.DisabledReason)
			for _, seg := range out.Segments {
				if len(hits) >= topK {
					break
				}
				if strings.TrimSpace(seg.Text) == "" {
					continue
				}
				artifactType := strings.TrimSpace(seg.ArtifactType)
				if filterType != "" && artifactType == "" {
					artifactType = string(filterType)
				}
				idx := strings.Index(seg.Text, q)
				snippet := ""
				if idx >= 0 {
					snippet = sliceAround(seg.Text, idx, len(q), 220)
				} else {
					snippet = sliceAround(seg.Text, 0, 0, 220)
				}
				hits = append(hits, hit{
					ArtifactType: artifactType,
					RefPath:      strings.TrimSpace(seg.RefPath),
					Snippet:      snippet,
					Score:        seg.Score,
					Source:       seg.Source,
				})
			}
		}
	} else {
		disabledReason = "retrieval engine not configured"
	}

	// 2) 纯字符串检索降级：在当前激活的 JSON 中做简单定位（对开发/离线环境更友好）
	push := func(tpe entity.ArtifactType, content json.RawMessage) {
		if len(content) == 0 || len(hits) >= topK {
			return
		}
		s := string(content)
		idx := strings.Index(s, q)
		if idx < 0 {
			return
		}
		snippet := sliceAround(s, idx, len(q), 160)
		hits = append(hits, hit{
			ArtifactType: string(tpe),
			Snippet:      snippet,
			Score:        0,
			Source:       "substring",
		})
	}

	if t.in != nil {
		if filterType == "" || filterType == entity.ArtifactTypeWorldview {
			push(entity.ArtifactTypeWorldview, t.in.CurrentWorldview)
		}
		if filterType == "" || filterType == entity.ArtifactTypeCharacters {
			push(entity.ArtifactTypeCharacters, t.in.CurrentCharacters)
		}
		if filterType == "" || filterType == entity.ArtifactTypeOutline {
			push(entity.ArtifactTypeOutline, t.in.CurrentOutline)
		}
		if filterType == "" || filterType == t.in.Type {
			push(t.in.Type, t.in.CurrentArtifactRaw)
		}
	}

	out := struct {
		Query          string `json:"query"`
		Type           string `json:"type,omitempty"`
		TopK           int    `json:"top_k"`
		DisabledReason string `json:"disabled_reason,omitempty"`
		Hits           []hit  `json:"hits"`
	}{
		Query:          q,
		Type:           strings.TrimSpace(args.Type),
		TopK:           topK,
		DisabledReason: strings.TrimSpace(disabledReason),
		Hits:           hits,
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

type projectGetBriefTool struct {
	in *ArtifactGenerateInput
}

func newProjectGetBriefTool(in *ArtifactGenerateInput) *projectGetBriefTool {
	return &projectGetBriefTool{in: in}
}

func (t *projectGetBriefTool) GetType() string { return toolNameProjectGetBrief }

func (t *projectGetBriefTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        toolNameProjectGetBrief,
		Desc:        "返回项目标题/简介与当前任务类型的简要信息。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *projectGetBriefTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	out := struct {
		ProjectTitle       string `json:"project_title"`
		ProjectDescription string `json:"project_description"`
		TaskType           string `json:"task_type"`
	}{
		ProjectTitle:       "",
		ProjectDescription: "",
		TaskType:           "",
	}
	if t.in != nil {
		out.ProjectTitle = strings.TrimSpace(t.in.ProjectTitle)
		out.ProjectDescription = strings.TrimSpace(t.in.ProjectDescription)
		out.TaskType = strings.TrimSpace(string(t.in.Type))
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func sliceAround(s string, idx int, matchLen int, maxLen int) string {
	if idx < 0 || idx > len(s) || maxLen <= 0 {
		return ""
	}
	start := idx - maxLen/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(s) {
		end = len(s)
	}
	if end-start < matchLen && end < len(s) {
		need := matchLen - (end - start)
		end += need
		if end > len(s) {
			end = len(s)
		}
	}
	snippet := s[start:end]
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.ReplaceAll(snippet, "\r", " ")
	snippet = strings.TrimSpace(snippet)
	return snippet
}
