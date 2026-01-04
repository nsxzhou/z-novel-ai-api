#!/bin/bash

# 配置
API_URL="http://localhost:8080/v1"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="admin-password-123"

# 1. 登录
echo "Logging in as admin..."
LOGIN_RESP=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$ADMIN_EMAIL\", \"password\": \"$ADMIN_PASSWORD\"}")

TOKEN=$(echo $LOGIN_RESP | grep -oE '"access_token":"[^"]+"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Login failed: $LOGIN_RESP"
  exit 1
fi
echo "Login successful."

# 2. 获取当前用户
echo "Getting current user info..."
curl -s -X GET "$API_URL/users/me" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 3. 创建新租户
echo "Creating a new tenant..."
NEW_TENANT_SLUG="test-tenant-$(date +%s)"
CREATE_TENANT_RESP=$(curl -s -X POST "$API_URL/tenants" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"Test Tenant\", \"slug\": \"$NEW_TENANT_SLUG\"}")

echo "Create tenant response: $CREATE_TENANT_RESP"
TENANT_ID=$(echo $CREATE_TENANT_RESP | grep -oE '"id":"[^"]+"' | cut -d'"' -f4)

# 4. 列出所有租户
echo "Listing all tenants..."
curl -s -X GET "$API_URL/tenants" -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 5. 在新租户中注册用户
echo "Registering a member in the new tenant..."
curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"member@test.com\", \"password\": \"password123\", \"name\": \"Test Member\", \"tenant_id\": \"$TENANT_ID\"}" | python3 -m json.tool

echo "Verification completed."
