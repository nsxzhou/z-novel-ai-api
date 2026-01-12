package artifact

import (
	"context"
	"fmt"

	appretrieval "z-novel-ai-api/internal/application/retrieval"
	wfmodel "z-novel-ai-api/internal/workflow/model"
	workflowpipeline "z-novel-ai-api/internal/workflow/pipeline"
	wfport "z-novel-ai-api/internal/workflow/port"
)

type ArtifactGenerator struct {
	pipeline *workflowpipeline.ArtifactPipeline
}

func NewArtifactGenerator(factory wfport.ChatModelFactory, retrievalEngine *appretrieval.Engine) *ArtifactGenerator {
	return &ArtifactGenerator{
		pipeline: workflowpipeline.NewArtifactPipeline(factory, retrievalEngine, artifactValidator{}, artifactJSONPatcher{}),
	}
}

func (g *ArtifactGenerator) Generate(ctx context.Context, in *wfmodel.ArtifactGenerateInput) (*wfmodel.ArtifactGenerateOutput, error) {
	if g == nil || g.pipeline == nil {
		return nil, fmt.Errorf("artifact workflow not configured")
	}
	return g.pipeline.Generate(ctx, in)
}

func (g *ArtifactGenerator) ScanConflicts(ctx context.Context, in *wfmodel.ArtifactConflictScanInput) (*wfmodel.ArtifactConflictScanOutput, error) {
	if g == nil || g.pipeline == nil {
		return nil, fmt.Errorf("artifact workflow not configured")
	}
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}
	return g.pipeline.ScanConflicts(ctx, in)
}
