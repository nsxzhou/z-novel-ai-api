package foundation

import (
	"fmt"
	"strings"
	"unicode/utf8"

	storymodel "z-novel-ai-api/internal/application/story/model"
	"z-novel-ai-api/internal/domain/entity"
)

type FoundationPlanValidationError struct {
	Issues []string
}

func (e FoundationPlanValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "foundation plan validation failed"
	}
	return "foundation plan validation failed: " + strings.Join(e.Issues, "; ")
}

// ValidateFoundationPlan 对 FoundationPlan 做强约束校验，避免脏数据落库。
func ValidateFoundationPlan(plan *storymodel.FoundationPlan) error {
	var issues []string
	if plan == nil {
		return FoundationPlanValidationError{Issues: []string{"plan is nil"}}
	}

	if plan.Version <= 0 {
		issues = append(issues, "version must be positive")
	}

	entityKeys := make(map[string]struct{}, len(plan.Entities))
	entityNames := make(map[string]struct{}, len(plan.Entities))
	for i := range plan.Entities {
		e := plan.Entities[i]
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
		} else {
			if _, ok := entityNames[name]; ok {
				issues = append(issues, path+".name duplicated: "+name)
			} else {
				entityNames[name] = struct{}{}
			}
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

	volumeKeys := make(map[string]struct{}, len(plan.Volumes))
	chapterKeys := make(map[string]struct{})
	for i := range plan.Volumes {
		v := plan.Volumes[i]
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

	for i := range plan.Relations {
		r := plan.Relations[i]
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

	if utf8.RuneCountInString(plan.Project.WorldBible) > 100000 {
		issues = append(issues, "project.world_bible too long")
	}
	if utf8.RuneCountInString(plan.Project.WritingStyle) > 2000 {
		issues = append(issues, "project.writing_style too long")
	}
	if utf8.RuneCountInString(plan.Project.POV) > 2000 {
		issues = append(issues, "project.pov too long")
	}
	if utf8.RuneCountInString(plan.Project.Genre) > 64 {
		issues = append(issues, "project.genre too long")
	}

	if len(issues) > 0 {
		return FoundationPlanValidationError{Issues: issues}
	}
	return nil
}

func isValidEntityType(t entity.StoryEntityType) bool {
	switch t {
	case entity.EntityTypeCharacter,
		entity.EntityTypeItem,
		entity.EntityTypeLocation,
		entity.EntityTypeOrganization,
		entity.EntityTypeConcept:
		return true
	default:
		return false
	}
}

func isValidEntityImportance(i entity.EntityImportance) bool {
	switch i {
	case entity.ImportanceProtagonist,
		entity.ImportanceMajor,
		entity.ImportanceSecondary,
		entity.ImportanceMinor:
		return true
	default:
		return false
	}
}

func isValidRelationType(t entity.RelationType) bool {
	switch t {
	case entity.RelationTypeFriend,
		entity.RelationTypeEnemy,
		entity.RelationTypeFamily,
		entity.RelationTypeLover,
		entity.RelationTypeSubordinate,
		entity.RelationTypeMentor,
		entity.RelationTypeRival,
		entity.RelationTypeAlly:
		return true
	default:
		return false
	}
}
