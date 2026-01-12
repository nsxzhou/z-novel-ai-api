package context

import (
	"fmt"
	"strings"
	"z-novel-ai-api/internal/application/story/storyutil"
)

const (
	rollingContextRecentKeep      = 6
	rollingContextTriggerTurns    = 12
	rollingContextSummaryMaxRunes = 2000
	rollingContextTurnMaxRunes    = 400
)

// RollingConversationContext 是“上下文滚动摘要”的 Redis 存储结构：
// - Summary：较早历史的压缩摘要（按需滚动追加，长度受限）
// - RecentUserTurns：最近若干条用户指令（用于保持短期连续性）
// - UserTurnCount：用于判断是否超过阈值，触发滚动压缩
type RollingConversationContext struct {
	Summary         string   `json:"summary"`
	RecentUserTurns []string `json:"recent_user_turns"`
	UserTurnCount   int      `json:"user_turn_count"`
}

func (c *RollingConversationContext) SnapshotForPrompt() (summary string, recentUserTurns string) {
	if c == nil {
		return "", ""
	}
	summary = strings.TrimSpace(c.Summary)
	recentUserTurns = formatRecentUserTurns(c.RecentUserTurns)
	return summary, recentUserTurns
}

func (c *RollingConversationContext) AppendUserPrompt(prompt string) {
	if c == nil {
		return
	}
	p := strings.TrimSpace(prompt)
	if p == "" {
		return
	}
	p = storyutil.TruncateByRunes(p, rollingContextTurnMaxRunes)

	c.UserTurnCount++
	c.RecentUserTurns = append(c.RecentUserTurns, p)
	c.compact()
}

func (c *RollingConversationContext) compact() {
	if c == nil {
		return
	}

	// 未超过阈值：保留更多“最近指令”，但限制上界，避免无限增长。
	if c.UserTurnCount <= rollingContextTriggerTurns {
		if len(c.RecentUserTurns) > rollingContextTriggerTurns {
			c.RecentUserTurns = c.RecentUserTurns[len(c.RecentUserTurns)-rollingContextTriggerTurns:]
		}
		return
	}

	// 超过阈值：滚动压缩，把较早的 recent turns 合并进 summary，只保留最后 N 条。
	if len(c.RecentUserTurns) <= rollingContextRecentKeep {
		return
	}

	older := c.RecentUserTurns[:len(c.RecentUserTurns)-rollingContextRecentKeep]
	c.RecentUserTurns = c.RecentUserTurns[len(c.RecentUserTurns)-rollingContextRecentKeep:]

	c.Summary = appendToSummary(c.Summary, older)
	c.Summary = storyutil.TruncateByRunes(c.Summary, rollingContextSummaryMaxRunes)
}

func formatRecentUserTurns(turns []string) string {
	if len(turns) == 0 {
		return ""
	}
	var b strings.Builder
	for i := range turns {
		t := strings.TrimSpace(turns[i])
		if t == "" {
			continue
		}
		_, _ = fmt.Fprintf(&b, "%d) %s\n", i+1, t)
	}
	return strings.TrimSpace(b.String())
}

func appendToSummary(summary string, older []string) string {
	var b strings.Builder
	s := strings.TrimSpace(summary)
	if s != "" {
		b.WriteString(s)
		b.WriteString("\n")
	}
	for i := range older {
		t := strings.TrimSpace(older[i])
		if t == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(t)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}
