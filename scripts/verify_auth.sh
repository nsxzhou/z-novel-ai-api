#!/bin/bash
set -euo pipefail

# 配置
API_URL="${API_URL:-http://localhost:8080/v1}"
TENANT_ID="${TENANT_ID:-}"

# 默认与 cmd/bootstrap/main.go 保持一致（可通过环境变量覆盖）
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@nsxzhou.fun}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

if [ -z "$TENANT_ID" ]; then
  echo "TENANT_ID is required."
  echo "Tip: run \"go run ./cmd/bootstrap\" and copy the printed default tenant ID."
  exit 1
fi

# 1. 登录
echo "Logging in as admin..."
LOGIN_RESP=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\", \"tenant_id\": \"$TENANT_ID\"}")

TOKEN=$(echo $LOGIN_RESP | grep -oE '"access_token":"[^"]+"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Login failed: $LOGIN_RESP"
  exit 1
fi
echo "Login successful."

# 2. 获取当前用户
echo "Getting current user info..."
curl -s -X GET "$API_URL/users/me" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 3. 打开当前租户公开注册（默认关闭）
echo "Enabling public registration for current tenant..."
curl -s -X PUT "$API_URL/tenants/current" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"settings\": {\"allow_public_registration\": true}}" | python3 -m json.tool

# 4. 列出所有租户（admin）
echo "Listing all tenants..."
curl -s -X GET "$API_URL/tenants" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 5. 在当前租户中注册用户（无需登录）
MEMBER_EMAIL="member-$(date +%s)@test.com"
MEMBER_PASSWORD="password123"
echo "Registering a member in the current tenant..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$MEMBER_EMAIL\", \"password\": \"$MEMBER_PASSWORD\", \"name\": \"Test Member\", \"tenant_id\": \"$TENANT_ID\"}" | python3 -m json.tool

# 6. 使用新用户登录验证
echo "Logging in as member..."
MEMBER_LOGIN_RESP=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$MEMBER_EMAIL\", \"password\": \"$MEMBER_PASSWORD\", \"tenant_id\": \"$TENANT_ID\"}")
MEMBER_TOKEN=$(echo $MEMBER_LOGIN_RESP | grep -oE '"access_token":"[^"]+"' | cut -d'"' -f4)
if [ -z "$MEMBER_TOKEN" ]; then
  echo "Member login failed: $MEMBER_LOGIN_RESP"
  exit 1
fi
echo "Member login successful."

echo "Verification completed."
