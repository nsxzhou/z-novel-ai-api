// Package postgres 提供 PostgreSQL Repository 实现
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/domain/repository"
)

// UserRepository 用户仓储实现
type UserRepository struct {
	client *Client
}

// NewUserRepository 创建用户仓储
func NewUserRepository(client *Client) *UserRepository {
	return &UserRepository{client: client}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Create")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(user.Settings)

	query := `
		INSERT INTO users (id, tenant_id, external_id, email, name, avatar_url, role, settings, last_login_at, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var externalID, avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	if user.ExternalID != "" {
		externalID = sql.NullString{String: user.ExternalID, Valid: true}
	}
	if user.AvatarURL != "" {
		avatarURL = sql.NullString{String: user.AvatarURL, Valid: true}
	}
	if user.LastLoginAt != nil {
		lastLoginAt = sql.NullTime{Time: *user.LastLoginAt, Valid: true}
	}

	err := q.QueryRowContext(ctx, query,
		user.TenantID, externalID, user.Email, user.Name, avatarURL, user.Role, settingsJSON, lastLoginAt,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取用户
func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, external_id, email, name, avatar_url, role, settings, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	return r.scanUser(q.QueryRowContext(ctx, query, id))
}

// GetByEmail 根据邮箱获取用户
func (r *UserRepository) GetByEmail(ctx context.Context, tenantID, email string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByEmail")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, external_id, email, name, avatar_url, role, settings, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1 AND email = $2
	`

	return r.scanUser(q.QueryRowContext(ctx, query, tenantID, email))
}

// GetByExternalID 根据外部 ID 获取用户
func (r *UserRepository) GetByExternalID(ctx context.Context, externalID string) (*entity.User, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.GetByExternalID")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `
		SELECT id, tenant_id, external_id, email, name, avatar_url, role, settings, last_login_at, created_at, updated_at
		FROM users
		WHERE external_id = $1
	`

	return r.scanUser(q.QueryRowContext(ctx, query, externalID))
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Update")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	settingsJSON, _ := json.Marshal(user.Settings)

	query := `
		UPDATE users
		SET name = $1, avatar_url = $2, role = $3, settings = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`

	err := q.QueryRowContext(ctx, query,
		user.Name, user.AvatarURL, user.Role, settingsJSON, user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.Delete")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `DELETE FROM users WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListByTenant 获取租户用户列表
func (r *UserRepository) ListByTenant(ctx context.Context, tenantID string, pagination repository.Pagination) (*repository.PagedResult[*entity.User], error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.ListByTenant")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	// 获取总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM users WHERE tenant_id = $1`
	if err := q.QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// 获取列表
	query := `
		SELECT id, tenant_id, external_id, email, name, avatar_url, role, settings, last_login_at, created_at, updated_at
		FROM users
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := q.QueryContext(ctx, query, tenantID, pagination.Limit(), pagination.Offset())
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			span.RecordError(err)
			return nil, err
		}
		users = append(users, user)
	}

	return repository.NewPagedResult(users, total, pagination), nil
}

// UpdateRole 更新用户角色
func (r *UserRepository) UpdateRole(ctx context.Context, id string, role entity.UserRole) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.UpdateRole")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`
	_, err := q.ExecContext(ctx, query, role, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.UpdateLastLogin")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	query := `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := q.ExecContext(ctx, query, id)

	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// ExistsByEmail 检查邮箱是否存在
func (r *UserRepository) ExistsByEmail(ctx context.Context, tenantID, email string) (bool, error) {
	ctx, span := tracer.Start(ctx, "postgres.UserRepository.ExistsByEmail")
	defer span.End()

	q := getQuerier(ctx, r.client.db)

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE tenant_id = $1 AND email = $2)`
	err := q.QueryRowContext(ctx, query, tenantID, email).Scan(&exists)

	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("failed to check email exists: %w", err)
	}

	return exists, nil
}

// scanUser 扫描单行用户数据
func (r *UserRepository) scanUser(row *sql.Row) (*entity.User, error) {
	var user entity.User
	var externalID, avatarURL sql.NullString
	var lastLoginAt sql.NullTime
	var settingsJSON []byte

	err := row.Scan(
		&user.ID, &user.TenantID, &externalID, &user.Email, &user.Name,
		&avatarURL, &user.Role, &settingsJSON, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	json.Unmarshal(settingsJSON, &user.Settings)

	return &user, nil
}

// scanUserFromRows 从多行结果扫描
func (r *UserRepository) scanUserFromRows(rows *sql.Rows) (*entity.User, error) {
	var user entity.User
	var externalID, avatarURL sql.NullString
	var lastLoginAt sql.NullTime
	var settingsJSON []byte

	err := rows.Scan(
		&user.ID, &user.TenantID, &externalID, &user.Email, &user.Name,
		&avatarURL, &user.Role, &settingsJSON, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan user row: %w", err)
	}

	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		user.LastLoginAt = &t
	}
	json.Unmarshal(settingsJSON, &user.Settings)

	return &user, nil
}
