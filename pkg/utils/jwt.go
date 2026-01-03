// Package utils 提供通用工具函数
package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims JWT 声明结构
type Claims struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Type     string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenPair 包含 AccessToken 和 RefreshToken
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// JWTManager JWT 管理器
type JWTManager struct {
	secret string
	issuer string
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secret, issuer string) *JWTManager {
	return &JWTManager{
		secret: secret,
		issuer: issuer,
	}
}

// GenerateTokenPair 生成双 Token
func (m *JWTManager) GenerateTokenPair(tenantID, userID, role string, accessTTL, refreshTTL time.Duration) (*TokenPair, error) {
	// 生成 AccessToken
	accessToken, err := m.GenerateToken(tenantID, userID, role, "access", accessTTL)
	if err != nil {
		return nil, err
	}

	// 生成 RefreshToken
	refreshToken, err := m.GenerateToken(tenantID, userID, role, "refresh", refreshTTL)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GenerateToken 生成单个 Token
func (m *JWTManager) GenerateToken(tenantID, userID, role, tokenType string, ttl time.Duration) (string, error) {
	claims := Claims{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		Type:     tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

// ParseToken 解析并验证 Token
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
