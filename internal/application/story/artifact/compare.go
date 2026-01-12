package artifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"z-novel-ai-api/internal/domain/entity"
)

type ArtifactCompareDiff struct {
	ArtifactType  string   `json:"artifact_type"`
	ChangedFields []string `json:"changed_fields,omitempty"`

	NovelFoundation *NovelFoundationCompareDiff `json:"novel_foundation,omitempty"`
	Worldview       *WorldviewCompareDiff       `json:"worldview,omitempty"`
	Characters      *CharactersCompareDiff      `json:"characters,omitempty"`
	Outline         *OutlineCompareDiff         `json:"outline,omitempty"`
}

type NovelFoundationCompareDiff struct {
	TitleChanged       bool `json:"title_changed"`
	DescriptionChanged bool `json:"description_changed"`
	GenreChanged       bool `json:"genre_changed"`
}

type WorldviewCompareDiff struct {
	GenreChanged           bool     `json:"genre_changed"`
	TargetWordCountChanged bool     `json:"target_word_count_changed"`
	WritingStyleChanged    bool     `json:"writing_style_changed"`
	POVChanged             bool     `json:"pov_changed"`
	TemperatureChanged     bool     `json:"temperature_changed"`
	WorldBibleChanged      bool     `json:"world_bible_changed"`
	WorldSettingsChanged   bool     `json:"world_settings_changed"`
	LocationsAdded         []string `json:"locations_added,omitempty"`
	LocationsRemoved       []string `json:"locations_removed,omitempty"`
}

type CharactersCompareDiff struct {
	EntitiesAdded   []string `json:"entities_added,omitempty"`
	EntitiesRemoved []string `json:"entities_removed,omitempty"`
	EntitiesUpdated []string `json:"entities_updated,omitempty"`

	RelationsAdded   []string `json:"relations_added,omitempty"`
	RelationsRemoved []string `json:"relations_removed,omitempty"`
	RelationsUpdated []string `json:"relations_updated,omitempty"`
}

type OutlineCompareDiff struct {
	VolumesAdded   []string `json:"volumes_added,omitempty"`
	VolumesRemoved []string `json:"volumes_removed,omitempty"`
	VolumesUpdated []string `json:"volumes_updated,omitempty"`

	ChaptersAdded   []string              `json:"chapters_added,omitempty"`
	ChaptersRemoved []string              `json:"chapters_removed,omitempty"`
	ChaptersUpdated []string              `json:"chapters_updated,omitempty"`
	ChaptersMoved   []ArtifactChapterMove `json:"chapters_moved,omitempty"`
}

type ArtifactChapterMove struct {
	Key           string `json:"key"`
	FromVolumeKey string `json:"from_volume_key"`
	ToVolumeKey   string `json:"to_volume_key"`
}

func CompareArtifactContent(t entity.ArtifactType, from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	if strings.TrimSpace(string(from)) == "" || strings.TrimSpace(string(to)) == "" {
		return &ArtifactCompareDiff{ArtifactType: string(t), ChangedFields: []string{"content"}}, nil
	}

	switch t {
	case entity.ArtifactTypeNovelFoundation:
		return compareNovelFoundation(from, to)
	case entity.ArtifactTypeWorldview:
		return compareWorldview(from, to)
	case entity.ArtifactTypeCharacters:
		return compareCharacters(from, to)
	case entity.ArtifactTypeOutline:
		return compareOutline(from, to)
	default:
		return compareByRawJSON(t, from, to)
	}
}

func compareByRawJSON(t entity.ArtifactType, from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	f := bytes.TrimSpace(from)
	u := bytes.TrimSpace(to)
	changed := !bytes.Equal(f, u)
	if !changed {
		return &ArtifactCompareDiff{ArtifactType: string(t)}, nil
	}
	return &ArtifactCompareDiff{ArtifactType: string(t), ChangedFields: []string{"content"}}, nil
}

func compareNovelFoundation(from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	var a, b NovelFoundationArtifact
	if err := json.Unmarshal(from, &a); err != nil {
		return compareByRawJSON(entity.ArtifactTypeNovelFoundation, from, to)
	}
	if err := json.Unmarshal(to, &b); err != nil {
		return compareByRawJSON(entity.ArtifactTypeNovelFoundation, from, to)
	}

	d := &NovelFoundationCompareDiff{}
	var changedFields []string

	if strings.TrimSpace(a.Title) != strings.TrimSpace(b.Title) {
		d.TitleChanged = true
		changedFields = append(changedFields, "title")
	}
	if strings.TrimSpace(a.Description) != strings.TrimSpace(b.Description) {
		d.DescriptionChanged = true
		changedFields = append(changedFields, "description")
	}
	if strings.TrimSpace(a.Genre) != strings.TrimSpace(b.Genre) {
		d.GenreChanged = true
		changedFields = append(changedFields, "genre")
	}

	out := &ArtifactCompareDiff{ArtifactType: string(entity.ArtifactTypeNovelFoundation)}
	if len(changedFields) > 0 {
		out.ChangedFields = changedFields
		out.NovelFoundation = d
	}
	return out, nil
}

func compareWorldview(from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	var a, b WorldviewArtifact
	if err := json.Unmarshal(from, &a); err != nil {
		return compareByRawJSON(entity.ArtifactTypeWorldview, from, to)
	}
	if err := json.Unmarshal(to, &b); err != nil {
		return compareByRawJSON(entity.ArtifactTypeWorldview, from, to)
	}

	d := &WorldviewCompareDiff{}
	var changedFields []string

	if strings.TrimSpace(a.Genre) != strings.TrimSpace(b.Genre) {
		d.GenreChanged = true
		changedFields = append(changedFields, "genre")
	}
	if a.TargetWordCount != b.TargetWordCount {
		d.TargetWordCountChanged = true
		changedFields = append(changedFields, "target_word_count")
	}
	if strings.TrimSpace(a.WritingStyle) != strings.TrimSpace(b.WritingStyle) {
		d.WritingStyleChanged = true
		changedFields = append(changedFields, "writing_style")
	}
	if strings.TrimSpace(a.POV) != strings.TrimSpace(b.POV) {
		d.POVChanged = true
		changedFields = append(changedFields, "pov")
	}
	if a.Temperature != b.Temperature {
		d.TemperatureChanged = true
		changedFields = append(changedFields, "temperature")
	}
	if strings.TrimSpace(a.WorldBible) != strings.TrimSpace(b.WorldBible) {
		d.WorldBibleChanged = true
		changedFields = append(changedFields, "world_bible")
	}

	wsChanged, locAdded, locRemoved := compareWorldSettings(a.WorldSettings, b.WorldSettings)
	if wsChanged {
		d.WorldSettingsChanged = true
		d.LocationsAdded = locAdded
		d.LocationsRemoved = locRemoved
		changedFields = append(changedFields, "world_settings")
	}

	out := &ArtifactCompareDiff{ArtifactType: string(entity.ArtifactTypeWorldview)}
	if len(changedFields) > 0 {
		out.ChangedFields = changedFields
		out.Worldview = d
	}
	return out, nil
}

func compareWorldSettings(a, b entity.WorldSettings) (changed bool, added []string, removed []string) {
	if strings.TrimSpace(a.TimeSystem) != strings.TrimSpace(b.TimeSystem) {
		changed = true
	}
	if strings.TrimSpace(a.Calendar) != strings.TrimSpace(b.Calendar) {
		changed = true
	}

	aLoc := normalizeStringSet(a.Locations)
	bLoc := normalizeStringSet(b.Locations)
	added, removed = diffStringSets(aLoc, bLoc)
	if len(added) > 0 || len(removed) > 0 {
		changed = true
	}
	return changed, added, removed
}

func compareCharacters(from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	var a, b CharactersArtifact
	if err := json.Unmarshal(from, &a); err != nil {
		return compareByRawJSON(entity.ArtifactTypeCharacters, from, to)
	}
	if err := json.Unmarshal(to, &b); err != nil {
		return compareByRawJSON(entity.ArtifactTypeCharacters, from, to)
	}

	d := &CharactersCompareDiff{}

	fromEnt := make(map[string]EntityPlan, len(a.Entities))
	toEnt := make(map[string]EntityPlan, len(b.Entities))
	for i := range a.Entities {
		k := strings.TrimSpace(a.Entities[i].Key)
		if k != "" {
			fromEnt[k] = a.Entities[i]
		}
	}
	for i := range b.Entities {
		k := strings.TrimSpace(b.Entities[i].Key)
		if k != "" {
			toEnt[k] = b.Entities[i]
		}
	}

	for k := range toEnt {
		if _, ok := fromEnt[k]; !ok {
			d.EntitiesAdded = append(d.EntitiesAdded, k)
		}
	}
	for k := range fromEnt {
		if _, ok := toEnt[k]; !ok {
			d.EntitiesRemoved = append(d.EntitiesRemoved, k)
		}
	}
	for k := range fromEnt {
		af, ok := fromEnt[k]
		if !ok {
			continue
		}
		bt, ok := toEnt[k]
		if !ok {
			continue
		}
		if !equalEntityPlan(af, bt) {
			d.EntitiesUpdated = append(d.EntitiesUpdated, k)
		}
	}

	fromRel := make(map[string]RelationPlan, len(a.Relations))
	toRel := make(map[string]RelationPlan, len(b.Relations))
	for i := range a.Relations {
		key := relationIdentity(a.Relations[i])
		if key != "" {
			fromRel[key] = a.Relations[i]
		}
	}
	for i := range b.Relations {
		key := relationIdentity(b.Relations[i])
		if key != "" {
			toRel[key] = b.Relations[i]
		}
	}

	for k := range toRel {
		if _, ok := fromRel[k]; !ok {
			d.RelationsAdded = append(d.RelationsAdded, k)
		}
	}
	for k := range fromRel {
		if _, ok := toRel[k]; !ok {
			d.RelationsRemoved = append(d.RelationsRemoved, k)
		}
	}
	for k := range fromRel {
		af, ok := fromRel[k]
		if !ok {
			continue
		}
		bt, ok := toRel[k]
		if !ok {
			continue
		}
		if !equalRelationPlan(af, bt) {
			d.RelationsUpdated = append(d.RelationsUpdated, k)
		}
	}

	sort.Strings(d.EntitiesAdded)
	sort.Strings(d.EntitiesRemoved)
	sort.Strings(d.EntitiesUpdated)
	sort.Strings(d.RelationsAdded)
	sort.Strings(d.RelationsRemoved)
	sort.Strings(d.RelationsUpdated)

	var changedFields []string
	if len(d.EntitiesAdded)+len(d.EntitiesRemoved)+len(d.EntitiesUpdated) > 0 {
		changedFields = append(changedFields, "entities")
	}
	if len(d.RelationsAdded)+len(d.RelationsRemoved)+len(d.RelationsUpdated) > 0 {
		changedFields = append(changedFields, "relations")
	}

	out := &ArtifactCompareDiff{ArtifactType: string(entity.ArtifactTypeCharacters)}
	if len(changedFields) > 0 {
		out.ChangedFields = changedFields
		out.Characters = d
	}
	return out, nil
}

func relationIdentity(r RelationPlan) string {
	sk := strings.TrimSpace(r.SourceKey)
	tk := strings.TrimSpace(r.TargetKey)
	rt := strings.TrimSpace(string(r.RelationType))
	if sk == "" || tk == "" || rt == "" {
		return ""
	}
	return fmt.Sprintf("%s->%s:%s", sk, tk, rt)
}

func equalEntityPlan(a, b EntityPlan) bool {
	if strings.TrimSpace(a.Name) != strings.TrimSpace(b.Name) {
		return false
	}
	if a.Type != b.Type {
		return false
	}
	if a.Importance != b.Importance {
		return false
	}
	if strings.TrimSpace(a.Description) != strings.TrimSpace(b.Description) {
		return false
	}
	if strings.TrimSpace(a.CurrentState) != strings.TrimSpace(b.CurrentState) {
		return false
	}

	if !equalStringSets(normalizeStringSet(a.Aliases), normalizeStringSet(b.Aliases)) {
		return false
	}

	if !equalEntityAttributes(a.Attributes, b.Attributes) {
		return false
	}
	return true
}

func equalEntityAttributes(a, b *entity.EntityAttributes) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = &entity.EntityAttributes{}
	}
	if b == nil {
		b = &entity.EntityAttributes{}
	}
	if a.Age != b.Age {
		return false
	}
	if strings.TrimSpace(a.Gender) != strings.TrimSpace(b.Gender) {
		return false
	}
	if strings.TrimSpace(a.Occupation) != strings.TrimSpace(b.Occupation) {
		return false
	}
	if strings.TrimSpace(a.Personality) != strings.TrimSpace(b.Personality) {
		return false
	}
	if strings.TrimSpace(a.Background) != strings.TrimSpace(b.Background) {
		return false
	}
	if !equalStringSets(normalizeStringSet(a.Abilities), normalizeStringSet(b.Abilities)) {
		return false
	}
	return true
}

func equalRelationPlan(a, b RelationPlan) bool {
	if strings.TrimSpace(a.SourceKey) != strings.TrimSpace(b.SourceKey) {
		return false
	}
	if strings.TrimSpace(a.TargetKey) != strings.TrimSpace(b.TargetKey) {
		return false
	}
	if a.RelationType != b.RelationType {
		return false
	}
	if a.Strength != b.Strength {
		return false
	}
	if strings.TrimSpace(a.Description) != strings.TrimSpace(b.Description) {
		return false
	}
	return equalRelationAttributes(a.Attributes, b.Attributes)
}

func equalRelationAttributes(a, b *entity.RelationAttributes) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil {
		a = &entity.RelationAttributes{}
	}
	if b == nil {
		b = &entity.RelationAttributes{}
	}
	if strings.TrimSpace(a.Since) != strings.TrimSpace(b.Since) {
		return false
	}
	if strings.TrimSpace(a.Origin) != strings.TrimSpace(b.Origin) {
		return false
	}
	if strings.TrimSpace(a.Development) != strings.TrimSpace(b.Development) {
		return false
	}
	return true
}

func compareOutline(from, to json.RawMessage) (*ArtifactCompareDiff, error) {
	var a, b OutlineArtifact
	if err := json.Unmarshal(from, &a); err != nil {
		return compareByRawJSON(entity.ArtifactTypeOutline, from, to)
	}
	if err := json.Unmarshal(to, &b); err != nil {
		return compareByRawJSON(entity.ArtifactTypeOutline, from, to)
	}

	d := &OutlineCompareDiff{}

	fromVol := make(map[string]VolumePlan, len(a.Volumes))
	toVol := make(map[string]VolumePlan, len(b.Volumes))
	for i := range a.Volumes {
		k := strings.TrimSpace(a.Volumes[i].Key)
		if k != "" {
			fromVol[k] = a.Volumes[i]
		}
	}
	for i := range b.Volumes {
		k := strings.TrimSpace(b.Volumes[i].Key)
		if k != "" {
			toVol[k] = b.Volumes[i]
		}
	}

	for k := range toVol {
		if _, ok := fromVol[k]; !ok {
			d.VolumesAdded = append(d.VolumesAdded, k)
		}
	}
	for k := range fromVol {
		if _, ok := toVol[k]; !ok {
			d.VolumesRemoved = append(d.VolumesRemoved, k)
		}
	}

	for k := range fromVol {
		av := fromVol[k]
		bv, ok := toVol[k]
		if !ok {
			continue
		}
		if strings.TrimSpace(av.Title) != strings.TrimSpace(bv.Title) || strings.TrimSpace(av.Summary) != strings.TrimSpace(bv.Summary) {
			d.VolumesUpdated = append(d.VolumesUpdated, k)
		}
	}

	chFrom, chFromVol := flattenChapters(a.Volumes)
	chTo, chToVol := flattenChapters(b.Volumes)

	for k := range chTo {
		if _, ok := chFrom[k]; !ok {
			d.ChaptersAdded = append(d.ChaptersAdded, k)
		}
	}
	for k := range chFrom {
		if _, ok := chTo[k]; !ok {
			d.ChaptersRemoved = append(d.ChaptersRemoved, k)
		}
	}
	for k := range chFrom {
		ca, ok := chFrom[k]
		if !ok {
			continue
		}
		cb, ok := chTo[k]
		if !ok {
			continue
		}
		if !equalChapterPlan(ca, cb) {
			d.ChaptersUpdated = append(d.ChaptersUpdated, k)
			continue
		}
		if strings.TrimSpace(chFromVol[k]) != strings.TrimSpace(chToVol[k]) {
			d.ChaptersMoved = append(d.ChaptersMoved, ArtifactChapterMove{
				Key:           k,
				FromVolumeKey: strings.TrimSpace(chFromVol[k]),
				ToVolumeKey:   strings.TrimSpace(chToVol[k]),
			})
		}
	}

	sort.Strings(d.VolumesAdded)
	sort.Strings(d.VolumesRemoved)
	sort.Strings(d.VolumesUpdated)
	sort.Strings(d.ChaptersAdded)
	sort.Strings(d.ChaptersRemoved)
	sort.Strings(d.ChaptersUpdated)
	sort.Slice(d.ChaptersMoved, func(i, j int) bool { return d.ChaptersMoved[i].Key < d.ChaptersMoved[j].Key })

	var changedFields []string
	if len(d.VolumesAdded)+len(d.VolumesRemoved)+len(d.VolumesUpdated)+len(d.ChaptersAdded)+len(d.ChaptersRemoved)+len(d.ChaptersUpdated)+len(d.ChaptersMoved) > 0 {
		changedFields = append(changedFields, "volumes")
	}

	out := &ArtifactCompareDiff{ArtifactType: string(entity.ArtifactTypeOutline)}
	if len(changedFields) > 0 {
		out.ChangedFields = changedFields
		out.Outline = d
	}
	return out, nil
}

func flattenChapters(vols []VolumePlan) (chapters map[string]ChapterPlan, volumeByChapter map[string]string) {
	chapters = make(map[string]ChapterPlan)
	volumeByChapter = make(map[string]string)
	for i := range vols {
		vk := strings.TrimSpace(vols[i].Key)
		for j := range vols[i].Chapters {
			ck := strings.TrimSpace(vols[i].Chapters[j].Key)
			if ck == "" {
				continue
			}
			chapters[ck] = vols[i].Chapters[j]
			volumeByChapter[ck] = vk
		}
	}
	return chapters, volumeByChapter
}

func equalChapterPlan(a, b ChapterPlan) bool {
	if strings.TrimSpace(a.Title) != strings.TrimSpace(b.Title) {
		return false
	}
	if strings.TrimSpace(a.Outline) != strings.TrimSpace(b.Outline) {
		return false
	}
	if a.TargetWordCount != b.TargetWordCount {
		return false
	}
	if a.StoryTimeStart != b.StoryTimeStart {
		return false
	}
	return true
}

func normalizeStringSet(in []string) []string {
	if len(in) == 0 {
		return nil
	}
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
	sort.Strings(out)
	return out
}

func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func diffStringSets(from, to []string) (added []string, removed []string) {
	af := make(map[string]struct{}, len(from))
	for i := range from {
		af[from[i]] = struct{}{}
	}
	bt := make(map[string]struct{}, len(to))
	for i := range to {
		bt[to[i]] = struct{}{}
	}
	for k := range bt {
		if _, ok := af[k]; !ok {
			added = append(added, k)
		}
	}
	for k := range af {
		if _, ok := bt[k]; !ok {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}
