package story

import (
	"context"
	"fmt"
	"strings"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

type FoundationApplyResult struct {
	ProjectUpdated bool `json:"project_updated"`

	EntitiesCreated  int `json:"entities_created"`
	EntitiesUpdated  int `json:"entities_updated"`
	RelationsCreated int `json:"relations_created"`
	RelationsUpdated int `json:"relations_updated"`
	VolumesCreated   int `json:"volumes_created"`
	VolumesUpdated   int `json:"volumes_updated"`
	ChaptersCreated  int `json:"chapters_created"`
	ChaptersUpdated  int `json:"chapters_updated"`
}

type FoundationApplier struct {
	projectRepo  repository.ProjectRepository
	entityRepo   repository.EntityRepository
	relationRepo repository.RelationRepository
	volumeRepo   repository.VolumeRepository
	chapterRepo  repository.ChapterRepository
}

func NewFoundationApplier(
	projectRepo repository.ProjectRepository,
	entityRepo repository.EntityRepository,
	relationRepo repository.RelationRepository,
	volumeRepo repository.VolumeRepository,
	chapterRepo repository.ChapterRepository,
) *FoundationApplier {
	return &FoundationApplier{
		projectRepo:  projectRepo,
		entityRepo:   entityRepo,
		relationRepo: relationRepo,
		volumeRepo:   volumeRepo,
		chapterRepo:  chapterRepo,
	}
}

// Apply 将 FoundationPlan 映射并落库到当前项目下。
//
// 约定：
// - 调用方负责事务边界（HTTP: DBTransaction 中间件；Worker: txMgr.WithTransaction + tenantCtx.SetTenant）。
// - 默认“追加/幂等”，不做破坏性删除。
func (a *FoundationApplier) Apply(ctx context.Context, projectID string, plan *FoundationPlan) (*FoundationApplyResult, error) {
	if a == nil {
		return nil, fmt.Errorf("foundation applier not configured")
	}
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	project, err := a.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	result := &FoundationApplyResult{}

	if changed := applyProjectPlan(project, &plan.Project); changed {
		if err := a.projectRepo.Update(ctx, project); err != nil {
			return nil, err
		}
		result.ProjectUpdated = true
	}

	entityIDByKey := make(map[string]string, len(plan.Entities))
	for i := range plan.Entities {
		p := plan.Entities[i]

		ent, created, updated, err := a.upsertEntity(ctx, projectID, &p)
		if err != nil {
			return nil, err
		}
		if created {
			result.EntitiesCreated++
		}
		if updated {
			result.EntitiesUpdated++
		}
		if ent != nil {
			entityIDByKey[p.Key] = ent.ID
		}
	}

	for i := range plan.Relations {
		rp := plan.Relations[i]
		srcID, ok := entityIDByKey[rp.SourceKey]
		if !ok {
			return nil, fmt.Errorf("relation source_key not found: %s", rp.SourceKey)
		}
		tgtID, ok := entityIDByKey[rp.TargetKey]
		if !ok {
			return nil, fmt.Errorf("relation target_key not found: %s", rp.TargetKey)
		}

		created, updated, err := a.upsertRelation(ctx, projectID, srcID, tgtID, &rp)
		if err != nil {
			return nil, err
		}
		if created {
			result.RelationsCreated++
		}
		if updated {
			result.RelationsUpdated++
		}
	}

	// 卷/章：按 ai_key 稳定映射，并按 plan 顺序重排 seq_num
	nextVolSeq, err := a.volumeRepo.GetNextSeqNum(ctx, projectID)
	if err != nil {
		return nil, err
	}
	volumeIDsInOrder := make([]string, 0, len(plan.Volumes))
	for i := range plan.Volumes {
		vp := plan.Volumes[i]

		vol, created, updated, err := a.upsertVolume(ctx, projectID, &nextVolSeq, &vp)
		if err != nil {
			return nil, err
		}
		if created {
			result.VolumesCreated++
		}
		if updated {
			result.VolumesUpdated++
		}
		if vol == nil {
			return nil, fmt.Errorf("volume upsert returned nil")
		}
		volumeIDsInOrder = append(volumeIDsInOrder, vol.ID)

		nextChSeq, err := a.chapterRepo.GetNextSeqNum(ctx, projectID, vol.ID)
		if err != nil {
			return nil, err
		}

		chapterIDsInOrder := make([]string, 0, len(vp.Chapters))
		for j := range vp.Chapters {
			cp := vp.Chapters[j]
			ch, created, updated, err := a.upsertChapter(ctx, projectID, vol.ID, &nextChSeq, &cp)
			if err != nil {
				return nil, err
			}
			if created {
				result.ChaptersCreated++
			}
			if updated {
				result.ChaptersUpdated++
			}
			if ch != nil {
				chapterIDsInOrder = append(chapterIDsInOrder, ch.ID)
			}
		}

		if err := a.chapterRepo.ReorderChapters(ctx, projectID, vol.ID, chapterIDsInOrder); err != nil {
			return nil, err
		}
	}

	if err := a.volumeRepo.ReorderVolumes(ctx, projectID, volumeIDsInOrder); err != nil {
		return nil, err
	}

	return result, nil
}

func applyProjectPlan(p *entity.Project, plan *ProjectPlan) (changed bool) {
	if p == nil || plan == nil {
		return false
	}

	if strings.TrimSpace(plan.Genre) != "" && p.Genre != strings.TrimSpace(plan.Genre) {
		p.Genre = strings.TrimSpace(plan.Genre)
		changed = true
	}
	if plan.TargetWordCount > 0 && p.TargetWordCount != plan.TargetWordCount {
		p.TargetWordCount = plan.TargetWordCount
		changed = true
	}

	if p.Settings == nil {
		p.Settings = &entity.ProjectSettings{}
		changed = true
	}
	if strings.TrimSpace(plan.WritingStyle) != "" && p.Settings.WritingStyle != strings.TrimSpace(plan.WritingStyle) {
		p.Settings.WritingStyle = strings.TrimSpace(plan.WritingStyle)
		changed = true
	}
	if strings.TrimSpace(plan.POV) != "" && p.Settings.POV != strings.TrimSpace(plan.POV) {
		p.Settings.POV = strings.TrimSpace(plan.POV)
		changed = true
	}
	if plan.Temperature != 0 && p.Settings.Temperature != plan.Temperature {
		p.Settings.Temperature = plan.Temperature
		changed = true
	}

	if p.WorldSettings == nil {
		p.WorldSettings = &entity.WorldSettings{}
		changed = true
	}
	// world_settings 一期按“整体覆盖”处理（Plan 强约束字段）；后续可细化为 merge。
	if !isEmptyWorldSettings(plan.WorldSettings) {
		*p.WorldSettings = plan.WorldSettings
		changed = true
	}

	if strings.TrimSpace(plan.WorldBible) != "" {
		next := upsertWorldBibleIntoDescription(p.Description, strings.TrimSpace(plan.WorldBible))
		if next != p.Description {
			p.Description = next
			changed = true
		}
	}

	return changed
}

func isEmptyWorldSettings(ws entity.WorldSettings) bool {
	if strings.TrimSpace(ws.TimeSystem) != "" {
		return false
	}
	if strings.TrimSpace(ws.Calendar) != "" {
		return false
	}
	return len(ws.Locations) == 0
}

func upsertWorldBibleIntoDescription(description, worldBible string) string {
	if strings.TrimSpace(worldBible) == "" {
		return description
	}
	if strings.Contains(description, worldBible) {
		return description
	}
	if strings.TrimSpace(description) == "" {
		return worldBible
	}
	return strings.TrimRight(description, "\n") + "\n\n---\n\n世界观设定：\n" + worldBible
}

func (a *FoundationApplier) upsertEntity(ctx context.Context, projectID string, p *EntityPlan) (*entity.StoryEntity, bool, bool, error) {
	if p == nil {
		return nil, false, false, nil
	}

	key := strings.TrimSpace(p.Key)
	if key == "" {
		return nil, false, false, fmt.Errorf("entity key is required")
	}

	name := strings.TrimSpace(p.Name)
	existing, err := a.entityRepo.GetByAIKey(ctx, projectID, key)
	if err != nil {
		return nil, false, false, err
	}

	if existing == nil {
		ent := entity.NewStoryEntity(projectID, name, p.Type, p.Importance)
		ent.AIKey = key
		ent.Description = strings.TrimSpace(p.Description)
		ent.CurrentState = strings.TrimSpace(p.CurrentState)
		if len(p.Aliases) > 0 {
			ent.Aliases = entity.StringSlice(uniqueStrings(p.Aliases))
		}
		if p.Attributes != nil {
			ent.Attributes = p.Attributes
		}
		if p.Importance != "" {
			ent.Importance = p.Importance
		}

		if err := a.entityRepo.Create(ctx, ent); err != nil {
			return nil, false, false, err
		}
		return ent, true, false, nil
	}

	if existing.Type != p.Type {
		return nil, false, false, fmt.Errorf("entity type mismatch for ai_key: key=%s existing=%s plan=%s", key, existing.Type, p.Type)
	}

	updated := false
	if existing.AIKey != key {
		existing.AIKey = key
		updated = true
	}
	if strings.TrimSpace(name) != "" && existing.Name != name {
		existing.Name = name
		updated = true
	}
	if strings.TrimSpace(p.Description) != "" && existing.Description != strings.TrimSpace(p.Description) {
		existing.Description = strings.TrimSpace(p.Description)
		updated = true
	}
	if strings.TrimSpace(p.CurrentState) != "" && existing.CurrentState != strings.TrimSpace(p.CurrentState) {
		existing.CurrentState = strings.TrimSpace(p.CurrentState)
		updated = true
	}
	if p.Attributes != nil {
		existing.Attributes = p.Attributes
		updated = true
	}
	if p.Importance != "" && existing.Importance != p.Importance {
		existing.Importance = p.Importance
		updated = true
	}

	if len(p.Aliases) > 0 {
		next := mergeAliases(existing.Aliases, p.Aliases)
		if len(next) != len(existing.Aliases) {
			existing.Aliases = next
			updated = true
		}
	}

	if updated {
		if err := a.entityRepo.Update(ctx, existing); err != nil {
			return nil, false, false, err
		}
	}

	return existing, false, updated, nil
}

func (a *FoundationApplier) upsertRelation(ctx context.Context, projectID, sourceID, targetID string, p *RelationPlan) (bool, bool, error) {
	if p == nil {
		return false, false, nil
	}

	existing, err := a.relationRepo.GetByEntitiesAndType(ctx, projectID, sourceID, targetID, p.RelationType)
	if err != nil {
		return false, false, err
	}

	if existing == nil {
		rel := entity.NewRelation(projectID, sourceID, targetID, p.RelationType)
		if p.Strength != 0 {
			rel.Strength = p.Strength
		}
		rel.Description = strings.TrimSpace(p.Description)
		if p.Attributes != nil {
			rel.Attributes = p.Attributes
		}
		if err := a.relationRepo.Create(ctx, rel); err != nil {
			return false, false, err
		}
		return true, false, nil
	}

	updated := false
	if p.Strength != 0 && existing.Strength != p.Strength {
		existing.Strength = p.Strength
		updated = true
	}
	if strings.TrimSpace(p.Description) != "" && existing.Description != strings.TrimSpace(p.Description) {
		existing.Description = strings.TrimSpace(p.Description)
		updated = true
	}
	if p.Attributes != nil {
		existing.Attributes = p.Attributes
		updated = true
	}

	if updated {
		if err := a.relationRepo.Update(ctx, existing); err != nil {
			return false, false, err
		}
	}
	return false, updated, nil
}

func (a *FoundationApplier) upsertVolume(ctx context.Context, projectID string, nextSeq *int, p *VolumePlan) (*entity.Volume, bool, bool, error) {
	if p == nil {
		return nil, false, false, nil
	}
	key := strings.TrimSpace(p.Key)
	if key == "" {
		return nil, false, false, fmt.Errorf("volume key is required")
	}

	existing, err := a.volumeRepo.GetByAIKey(ctx, projectID, key)
	if err != nil {
		return nil, false, false, err
	}
	if existing == nil {
		seqNum := 0
		if nextSeq != nil {
			seqNum = *nextSeq
			*nextSeq = *nextSeq + 1
		} else {
			seqNum, err = a.volumeRepo.GetNextSeqNum(ctx, projectID)
			if err != nil {
				return nil, false, false, err
			}
		}
		vol := entity.NewVolume(projectID, seqNum, strings.TrimSpace(p.Title))
		vol.AIKey = key
		vol.Summary = strings.TrimSpace(p.Summary)
		if err := a.volumeRepo.Create(ctx, vol); err != nil {
			return nil, false, false, err
		}
		return vol, true, false, nil
	}

	updated := false
	if existing.AIKey != key {
		existing.AIKey = key
		updated = true
	}
	if strings.TrimSpace(p.Title) != "" && existing.Title != strings.TrimSpace(p.Title) {
		existing.Title = strings.TrimSpace(p.Title)
		updated = true
	}
	if strings.TrimSpace(p.Summary) != "" && existing.Summary != strings.TrimSpace(p.Summary) {
		existing.Summary = strings.TrimSpace(p.Summary)
		updated = true
	}

	if updated {
		if err := a.volumeRepo.Update(ctx, existing); err != nil {
			return nil, false, false, err
		}
	}
	return existing, false, updated, nil
}

func (a *FoundationApplier) upsertChapter(ctx context.Context, projectID, volumeID string, nextSeq *int, p *ChapterPlan) (*entity.Chapter, bool, bool, error) {
	if p == nil {
		return nil, false, false, nil
	}
	key := strings.TrimSpace(p.Key)
	if key == "" {
		return nil, false, false, fmt.Errorf("chapter key is required")
	}

	existing, err := a.chapterRepo.GetByAIKey(ctx, projectID, key)
	if err != nil {
		return nil, false, false, err
	}
	if existing == nil {
		seqNum := 0
		if nextSeq != nil {
			seqNum = *nextSeq
			*nextSeq = *nextSeq + 1
		} else {
			seqNum, err = a.chapterRepo.GetNextSeqNum(ctx, projectID, volumeID)
			if err != nil {
				return nil, false, false, err
			}
		}
		ch := entity.NewChapter(projectID, volumeID, seqNum)
		ch.AIKey = key
		ch.Title = strings.TrimSpace(p.Title)
		ch.Outline = strings.TrimSpace(p.Outline)
		ch.StoryTimeStart = p.StoryTimeStart
		ch.Notes = upsertTargetWordCountLine(ch.Notes, p.TargetWordCount)
		if err := a.chapterRepo.Create(ctx, ch); err != nil {
			return nil, false, false, err
		}
		return ch, true, false, nil
	}

	updated := false
	if existing.AIKey != key {
		existing.AIKey = key
		updated = true
	}
	if strings.TrimSpace(volumeID) != "" && existing.VolumeID != volumeID {
		existing.VolumeID = volumeID
		if nextSeq != nil {
			existing.SeqNum = *nextSeq
			*nextSeq = *nextSeq + 1
		}
		updated = true
	}
	if strings.TrimSpace(p.Title) != "" && existing.Title != strings.TrimSpace(p.Title) {
		existing.Title = strings.TrimSpace(p.Title)
		updated = true
	}
	if strings.TrimSpace(p.Outline) != "" && existing.Outline != strings.TrimSpace(p.Outline) {
		existing.Outline = strings.TrimSpace(p.Outline)
		updated = true
	}
	if p.StoryTimeStart != 0 && existing.StoryTimeStart != p.StoryTimeStart {
		existing.StoryTimeStart = p.StoryTimeStart
		updated = true
	}

	if p.TargetWordCount > 0 {
		nextNotes := upsertTargetWordCountLine(existing.Notes, p.TargetWordCount)
		if nextNotes != existing.Notes {
			existing.Notes = nextNotes
			updated = true
		}
	}

	if updated {
		if err := a.chapterRepo.Update(ctx, existing); err != nil {
			return nil, false, false, err
		}
	}
	return existing, false, updated, nil
}

func upsertTargetWordCountLine(notes string, target int) string {
	if target <= 0 {
		return notes
	}
	lines := strings.Split(notes, "\n")
	prefix := "target_word_count:"
	found := false
	for i := range lines {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), prefix) {
			lines[i] = fmt.Sprintf("%s %d", prefix, target)
			found = true
		}
	}
	if !found {
		if strings.TrimSpace(notes) == "" {
			return fmt.Sprintf("%s %d", prefix, target)
		}
		lines = append(lines, fmt.Sprintf("%s %d", prefix, target))
	}
	return strings.Join(lines, "\n")
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for i := range in {
		s := strings.TrimSpace(in[i])
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func mergeAliases(existing entity.StringSlice, incoming []string) entity.StringSlice {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	out := make([]string, 0, len(existing)+len(incoming))
	for i := range existing {
		s := strings.TrimSpace(existing[i])
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for i := range incoming {
		s := strings.TrimSpace(incoming[i])
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return entity.StringSlice(out)
}
