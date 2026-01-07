// Package entity 定义领域实体
package entity

// Role 对话角色枚举
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)
