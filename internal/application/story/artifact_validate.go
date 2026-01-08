package story

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"z-novel-ai-api/internal/domain/entity"
)

type NovelFoundationArtifact struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Genre       string `json:"genre,omitempty"`
}

type WorldviewArtifact struct {
	Genre           string               `json:"genre,omitempty"`
	TargetWordCount int                  `json:"target_word_count,omitempty"`
	WritingStyle    string               `json:"writing_style,omitempty"`
	POV             string               `json:"pov,omitempty"`
	Temperature     float64              `json:"temperature,omitempty"`
	WorldSettings   entity.WorldSettings `json:"world_settings"`
	WorldBible      string               `json:"world_bible,omitempty"`
}

type CharactersArtifact struct {
	Entities  []EntityPlan   `json:"entities"`
	Relations []RelationPlan `json:"relations"`
}

type OutlineArtifact struct {
	Volumes []VolumePlan `json:"volumes"`
}

type ArtifactValidationError struct {
	Type   entity.ArtifactType
	Issues []string
}

func (e ArtifactValidationError) Error() string {
	if len(e.Issues) == 0 {
		return fmt.Sprintf("artifact validation failed: %s", e.Type)
	}
	return fmt.Sprintf("artifact validation failed: %s: %s", e.Type, strings.Join(e.Issues, "; "))
}

func normalizeAndValidateArtifact(t entity.ArtifactType, rawJSON string) (json.RawMessage, error) {
	switch t {
	case entity.ArtifactTypeNovelFoundation:
		var a NovelFoundationArtifact
		if err := json.Unmarshal([]byte(rawJSON), &a); err != nil {
			return nil, fmt.Errorf("failed to parse novel_foundation json: %w", err)
		}
		if err := ValidateNovelFoundationArtifact(&a); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(a)
		return b, nil

	case entity.ArtifactTypeWorldview:
		var a WorldviewArtifact
		if err := json.Unmarshal([]byte(rawJSON), &a); err != nil {
			return nil, fmt.Errorf("failed to parse worldview json: %w", err)
		}
		if err := ValidateWorldviewArtifact(&a); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(a)
		return b, nil

	case entity.ArtifactTypeCharacters:
		var a CharactersArtifact
		if err := json.Unmarshal([]byte(rawJSON), &a); err != nil {
			return nil, fmt.Errorf("failed to parse characters json: %w", err)
		}
		if err := ValidateCharactersArtifact(&a); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(a)
		return b, nil

	case entity.ArtifactTypeOutline:
		var a OutlineArtifact
		if err := json.Unmarshal([]byte(rawJSON), &a); err != nil {
			return nil, fmt.Errorf("failed to parse outline json: %w", err)
		}
		if err := ValidateOutlineArtifact(&a); err != nil {
			return nil, err
		}
		b, _ := json.Marshal(a)
		return b, nil

	default:
		return nil, fmt.Errorf("unsupported artifact type: %s", t)
	}
}

func ValidateNovelFoundationArtifact(a *NovelFoundationArtifact) error {
	var issues []string
	if a == nil {
		return ArtifactValidationError{Type: entity.ArtifactTypeNovelFoundation, Issues: []string{"artifact is nil"}}
	}
	if strings.TrimSpace(a.Title) == "" {
		issues = append(issues, "title is required")
	} else if utf8.RuneCountInString(a.Title) > 255 {
		issues = append(issues, "title too long")
	}
	if strings.TrimSpace(a.Description) == "" {
		issues = append(issues, "description is required")
	} else if utf8.RuneCountInString(a.Description) > 50000 {
		issues = append(issues, "description too long")
	}
	if utf8.RuneCountInString(a.Genre) > 64 {
		issues = append(issues, "genre too long")
	}
	if len(issues) > 0 {
		return ArtifactValidationError{Type: entity.ArtifactTypeNovelFoundation, Issues: issues}
	}
	return nil
}

func ValidateWorldviewArtifact(a *WorldviewArtifact) error {
	var issues []string
	if a == nil {
		return ArtifactValidationError{Type: entity.ArtifactTypeWorldview, Issues: []string{"artifact is nil"}}
	}
	if utf8.RuneCountInString(a.WorldBible) > 200000 {
		issues = append(issues, "world_bible too long")
	}
	if utf8.RuneCountInString(a.WritingStyle) > 5000 {
		issues = append(issues, "writing_style too long")
	}
	if utf8.RuneCountInString(a.POV) > 5000 {
		issues = append(issues, "pov too long")
	}
	if utf8.RuneCountInString(a.Genre) > 64 {
		issues = append(issues, "genre too long")
	}
	if isEmptyWorldSettings(a.WorldSettings) {
		issues = append(issues, "world_settings is required")
	}
	if len(issues) > 0 {
		return ArtifactValidationError{Type: entity.ArtifactTypeWorldview, Issues: issues}
	}
	return nil
}

func ValidateCharactersArtifact(a *CharactersArtifact) error {
	var issues []string
	if a == nil {
		return ArtifactValidationError{Type: entity.ArtifactTypeCharacters, Issues: []string{"artifact is nil"}}
	}

	entityKeys := make(map[string]struct{}, len(a.Entities))
	for i := range a.Entities {
		e := a.Entities[i]
		path := fmt.Sprintf("entities[%d]", i)

		key := strings.TrimSpace(e.Key)
		if key == "" {
			issues = append(issues, path+".key is required")
		} else {
			if _, ok := entityKeys[key]; ok {
				issues = append(issues, path+".key duplicated: "+key)
			} else {
				entityKeys[key] = struct{}{}
			}
			if utf8.RuneCountInString(key) > 128 {
				issues = append(issues, path+".key too long")
			}
		}

		name := strings.TrimSpace(e.Name)
		if name == "" {
			issues = append(issues, path+".name is required")
		} else if utf8.RuneCountInString(name) > 128 {
			issues = append(issues, path+".name too long")
		}

		if !isValidEntityType(e.Type) {
			issues = append(issues, path+".type invalid: "+string(e.Type))
		}
		if e.Importance != "" && !isValidEntityImportance(e.Importance) {
			issues = append(issues, path+".importance invalid: "+string(e.Importance))
		}

		if utf8.RuneCountInString(e.Description) > 20000 {
			issues = append(issues, path+".description too long")
		}
		if utf8.RuneCountInString(e.CurrentState) > 20000 {
			issues = append(issues, path+".current_state too long")
		}
	}

	for i := range a.Relations {
		r := a.Relations[i]
		path := fmt.Sprintf("relations[%d]", i)

		sk := strings.TrimSpace(r.SourceKey)
		tk := strings.TrimSpace(r.TargetKey)
		if sk == "" {
			issues = append(issues, path+".source_key is required")
		}
		if tk == "" {
			issues = append(issues, path+".target_key is required")
		}
		if sk != "" {
			if _, ok := entityKeys[sk]; !ok {
				issues = append(issues, path+".source_key not found: "+sk)
			}
		}
		if tk != "" {
			if _, ok := entityKeys[tk]; !ok {
				issues = append(issues, path+".target_key not found: "+tk)
			}
		}

		if !isValidRelationType(r.RelationType) {
			issues = append(issues, path+".relation_type invalid: "+string(r.RelationType))
		}

		if utf8.RuneCountInString(r.Description) > 20000 {
			issues = append(issues, path+".description too long")
		}
	}

	if len(issues) > 0 {
		return ArtifactValidationError{Type: entity.ArtifactTypeCharacters, Issues: issues}
	}
	return nil
}

func ValidateOutlineArtifact(a *OutlineArtifact) error {
	var issues []string
	if a == nil {
		return ArtifactValidationError{Type: entity.ArtifactTypeOutline, Issues: []string{"artifact is nil"}}
	}

	volumeKeys := make(map[string]struct{}, len(a.Volumes))
	chapterKeys := make(map[string]struct{})
	for i := range a.Volumes {
		v := a.Volumes[i]
		vPath := fmt.Sprintf("volumes[%d]", i)

		vKey := strings.TrimSpace(v.Key)
		if vKey == "" {
			issues = append(issues, vPath+".key is required")
		} else {
			if _, ok := volumeKeys[vKey]; ok {
				issues = append(issues, vPath+".key duplicated: "+vKey)
			} else {
				volumeKeys[vKey] = struct{}{}
			}
			if utf8.RuneCountInString(vKey) > 128 {
				issues = append(issues, vPath+".key too long")
			}
		}

		if strings.TrimSpace(v.Title) == "" {
			issues = append(issues, vPath+".title is required")
		} else if utf8.RuneCountInString(v.Title) > 255 {
			issues = append(issues, vPath+".title too long")
		}

		if utf8.RuneCountInString(v.Summary) > 20000 {
			issues = append(issues, vPath+".summary too long")
		}

		for j := range v.Chapters {
			ch := v.Chapters[j]
			cPath := fmt.Sprintf("%s.chapters[%d]", vPath, j)

			cKey := strings.TrimSpace(ch.Key)
			if cKey == "" {
				issues = append(issues, cPath+".key is required")
			} else {
				if _, ok := chapterKeys[cKey]; ok {
					issues = append(issues, cPath+".key duplicated: "+cKey)
				} else {
					chapterKeys[cKey] = struct{}{}
				}
				if utf8.RuneCountInString(cKey) > 128 {
					issues = append(issues, cPath+".key too long")
				}
			}

			if strings.TrimSpace(ch.Title) == "" {
				issues = append(issues, cPath+".title is required")
			} else if utf8.RuneCountInString(ch.Title) > 255 {
				issues = append(issues, cPath+".title too long")
			}

			if strings.TrimSpace(ch.Outline) == "" {
				issues = append(issues, cPath+".outline is required")
			} else if utf8.RuneCountInString(ch.Outline) > 50000 {
				issues = append(issues, cPath+".outline too long")
			}
		}
	}

	if len(issues) > 0 {
		return ArtifactValidationError{Type: entity.ArtifactTypeOutline, Issues: issues}
	}
	return nil
}
