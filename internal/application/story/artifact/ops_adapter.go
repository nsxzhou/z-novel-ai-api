package artifact

import (
	"context"
	"encoding/json"

	"z-novel-ai-api/internal/domain/entity"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	wfnode "z-novel-ai-api/internal/workflow/node"
)

type artifactValidator struct{}

func (artifactValidator) NormalizeAndValidate(t entity.ArtifactType, rawJSON string) (json.RawMessage, error) {
	return normalizeAndValidateArtifact(t, rawJSON)
}

type artifactJSONPatcher struct{}

func (artifactJSONPatcher) IsEnabled(in *wfmodel.ArtifactGenerateInput) bool {
	return isArtifactJSONPatchEnabled(in)
}

func (artifactJSONPatcher) AllowedOps() []string {
	return artifactJSONPatchAllowedOps()
}

func (artifactJSONPatcher) AllowedPaths(t entity.ArtifactType) []string {
	return artifactJSONPatchAllowedPaths(t)
}

func (artifactJSONPatcher) Apply(_ context.Context, t entity.ArtifactType, base json.RawMessage, patchText string) (json.RawMessage, error) {
	return applyArtifactJSONPatch(t, base, patchText)
}

var _ wfnode.ArtifactValidator = (*artifactValidator)(nil)
var _ wfnode.ArtifactJSONPatcher = (*artifactJSONPatcher)(nil)
