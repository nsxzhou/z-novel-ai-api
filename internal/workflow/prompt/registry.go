package prompt

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	einoprompt "github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

//go:embed templates/*.txt
var templatesFS embed.FS

type PromptID string

const (
	PromptFoundationPlanV1       PromptID = "foundation_plan_v1"
	PromptChapterGenV1           PromptID = "chapter_gen_v1"
	PromptArtifactV1             PromptID = "artifact_v1"
	PromptArtifactV2             PromptID = "artifact_v2"
	PromptArtifactPatchV1        PromptID = "artifact_patch_v1"
	PromptArtifactConflictScanV1 PromptID = "artifact_conflict_scan_v1"
	PromptProjectCreationV1      PromptID = "project_creation_v1"
)

type Registry struct {
	mu    sync.RWMutex
	cache map[PromptID]einoprompt.ChatTemplate
}

func NewRegistry() *Registry {
	return &Registry{
		cache: make(map[PromptID]einoprompt.ChatTemplate),
	}
}

func (r *Registry) ChatTemplate(id PromptID) (einoprompt.ChatTemplate, error) {
	if r == nil {
		return nil, fmt.Errorf("prompt registry is nil")
	}

	r.mu.RLock()
	if tpl, ok := r.cache[id]; ok {
		r.mu.RUnlock()
		return tpl, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if tpl, ok := r.cache[id]; ok {
		return tpl, nil
	}

	systemPath, userPath, err := resolvePromptFiles(id)
	if err != nil {
		return nil, err
	}
	system, err := readEmbeddedText(systemPath)
	if err != nil {
		return nil, err
	}
	user, err := readEmbeddedText(userPath)
	if err != nil {
		return nil, err
	}

	tpl := einoprompt.FromMessages(
		schema.FString,
		schema.SystemMessage(system),
		schema.UserMessage(user),
	)
	r.cache[id] = tpl
	return tpl, nil
}

func resolvePromptFiles(id PromptID) (systemFile string, userFile string, err error) {
	switch id {
	case PromptFoundationPlanV1:
		return "templates/foundation_plan_v1.system.txt", "templates/foundation_plan_v1.user.txt", nil
	case PromptChapterGenV1:
		return "templates/chapter_gen_v1.system.txt", "templates/chapter_gen_v1.user.txt", nil
	case PromptArtifactV1:
		return "templates/artifact_v1.system.txt", "templates/artifact_v1.user.txt", nil
	case PromptArtifactV2:
		return "templates/artifact_v2.system.txt", "templates/artifact_v2.user.txt", nil
	case PromptArtifactPatchV1:
		return "templates/artifact_patch_v1.system.txt", "templates/artifact_patch_v1.user.txt", nil
	case PromptArtifactConflictScanV1:
		return "templates/artifact_conflict_scan_v1.system.txt", "templates/artifact_conflict_scan_v1.user.txt", nil
	case PromptProjectCreationV1:
		return "templates/project_creation_v1.system.txt", "templates/project_creation_v1.user.txt", nil
	default:
		return "", "", fmt.Errorf("unknown prompt id: %s", id)
	}
}

func readEmbeddedText(path string) (string, error) {
	b, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
