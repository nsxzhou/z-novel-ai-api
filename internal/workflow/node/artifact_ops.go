package node

import (
	"context"
	"encoding/json"

	"z-novel-ai-api/internal/domain/entity"
	wfmodel "z-novel-ai-api/internal/workflow/model"
)

type ArtifactValidator interface {
	NormalizeAndValidate(t entity.ArtifactType, rawJSON string) (json.RawMessage, error)
}

type ArtifactJSONPatcher interface {
	IsEnabled(in *wfmodel.ArtifactGenerateInput) bool
	AllowedOps() []string
	AllowedPaths(t entity.ArtifactType) []string
	Apply(ctx context.Context, t entity.ArtifactType, base json.RawMessage, patchText string) (json.RawMessage, error)
}
