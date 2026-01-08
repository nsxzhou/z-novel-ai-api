package story

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"

	"z-novel-ai-api/internal/domain/entity"
)

func isArtifactJSONPatchEnabled(in *ArtifactGenerateInput) bool {
	if in == nil {
		return false
	}
	switch in.Type {
	case entity.ArtifactTypeNovelFoundation, entity.ArtifactTypeWorldview, entity.ArtifactTypeCharacters, entity.ArtifactTypeOutline:
		return len(bytes.TrimSpace(in.CurrentArtifactRaw)) > 0
	default:
		return false
	}
}

func artifactJSONPatchAllowedOps() []string {
	return []string{"add", "replace"}
}

func artifactJSONPatchAllowedPaths(t entity.ArtifactType) []string {
	switch t {
	case entity.ArtifactTypeNovelFoundation:
		return []string{"/title", "/description", "/genre"}
	case entity.ArtifactTypeWorldview:
		return []string{"/genre", "/target_word_count", "/writing_style", "/pov", "/temperature", "/world_bible", "/world_settings"}
	case entity.ArtifactTypeCharacters:
		return []string{"/entities", "/relations"}
	case entity.ArtifactTypeOutline:
		return []string{"/volumes"}
	default:
		return nil
	}
}

type jsonPatchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

func applyArtifactJSONPatch(t entity.ArtifactType, base json.RawMessage, patchText string) (json.RawMessage, error) {
	patchText = strings.TrimSpace(patchText)
	if patchText == "" {
		return nil, fmt.Errorf("empty json patch")
	}

	allowedPaths := make(map[string]struct{})
	for _, p := range artifactJSONPatchAllowedPaths(t) {
		allowedPaths[p] = struct{}{}
	}
	if len(allowedPaths) == 0 {
		return nil, fmt.Errorf("json patch not supported for artifact type: %s", t)
	}

	var ops []jsonPatchOp
	if err := json.Unmarshal([]byte(patchText), &ops); err != nil {
		return nil, fmt.Errorf("invalid json patch: %w", err)
	}
	if len(ops) == 0 {
		// 允许空 patch：表示“不变更”，仍返回当前 JSON。
		if len(bytes.TrimSpace(base)) == 0 {
			return json.RawMessage([]byte("{}")), nil
		}
		return base, nil
	}

	for i := range ops {
		op := strings.ToLower(strings.TrimSpace(ops[i].Op))
		if op != "add" && op != "replace" {
			return nil, fmt.Errorf("invalid json patch op at index %d: op=%s", i, strings.TrimSpace(ops[i].Op))
		}
		path := strings.TrimSpace(ops[i].Path)
		if _, ok := allowedPaths[path]; !ok {
			return nil, fmt.Errorf("invalid json patch path at index %d: path=%s", i, path)
		}
		if len(bytes.TrimSpace(ops[i].Value)) == 0 {
			return nil, fmt.Errorf("invalid json patch op at index %d: value is required", i)
		}
	}

	doc := bytes.TrimSpace(base)
	if len(doc) == 0 {
		doc = []byte("{}")
	}

	p, err := jsonpatch.DecodePatch([]byte(patchText))
	if err != nil {
		return nil, fmt.Errorf("invalid json patch: %w", err)
	}
	out, err := p.Apply(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to apply json patch: %w", err)
	}
	return out, nil
}
