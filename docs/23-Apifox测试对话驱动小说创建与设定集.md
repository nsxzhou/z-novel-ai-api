# 23 - Apifox 测试：对话驱动小说创建与设定集

更新时间：2026-01-06

> 目标：用 Apifox 完整跑通三条链路：
>
> 1. 对话式项目孵化（从模糊想法到正式项目）
> 2. 长期会话按任务切换生成设定集（worldview/characters/outline）
> 3. Foundation 一期 Plan→Apply 落库

---

## 0. 前置条件

### 0.1 服务已启动

- PostgreSQL + Redis 已就绪：`docker compose up -d`
- 迁移已执行：`make migrate-up`
- API Gateway 已启动：`JWT_SECRET="dev-secret" FEATURES_CORE_ENABLED=false go run ./cmd/api-gateway`

### 0.2 已有可登录用户与 tenant_id

执行 bootstrap 创建默认租户与管理员：`go run ./cmd/bootstrap`

记录输出中的：

- `tenant_id`
- `admin email/password`

### 0.3 LLM 配置可用

确保在 `configs/config.yaml` 或环境变量中配置了 `llm.providers.*` 的 `api_key/base_url/model`。

---

## 1. Apifox 环境与全局配置

### 1.1 环境变量

- `base_url`：如 `http://127.0.0.1:8080`
- `tenant_id`：bootstrap 输出的 UUID
- `email` / `password`：登录账号
- `access_token`：登录后写入
- `project_id`：创建项目后写入
- `session_id`：创建会话后写入
- `pc_session_id`：项目孵化会话 ID

### 1.2 全局 Header

- `Content-Type: application/json`
- `Authorization: Bearer {{access_token}}`

---

## 2. 用例 A：对话式项目孵化（新功能）

> 说明：无需先创建 Project，通过对话引导从模糊想法转化为正式项目。

### 2.1 登录获取 access_token

**请求**

- `POST {{base_url}}/v1/auth/login`

Body：

```json
{
  "tenant_id": "{{tenant_id}}",
  "email": "{{email}}",
  "password": "{{password}}"
}
```

**后置脚本**

```js
const res = pm.response.json();
pm.environment.set("access_token", res.data.access_token);
```

### 2.2 创建项目孵化会话

**请求**

- `POST {{base_url}}/v1/project-creation-sessions`

Body（可为空或携带初始 prompt）：

```json
{}
```

**断言**

- HTTP 201
- `data.id` 非空
- `data.stage == "discover"`

**后置脚本**

```js
const res = pm.response.json();
pm.environment.set("pc_session_id", res.data.id);
```

### 2.3 发送消息进入探索阶段

**请求**

- `POST {{base_url}}/v1/project-creation-sessions/{{pc_session_id}}/messages`

Body：

```json
{
  "prompt": "我想写一个关于记忆篡改的悬疑故事，但还没想好具体情节"
}
```

**断言**

- HTTP 200
- `data.session.stage` 为 `discover` 或 `narrow`（取决于 AI 判断）
- `data.assistant_turn_id` 非空

### 2.4 继续对话细化想法

**请求**

- `POST {{base_url}}/v1/project-creation-sessions/{{pc_session_id}}/messages`

Body：

```json
{
  "prompt": "主角是一个数据取证工程师，被卷入一场关于记忆篡改的阴谋。设定在近未来的城市。"
}
```

**断言**

- `data.session.stage` 进入 `narrow` 或 `draft`

### 2.5 确认创建项目

**请求**

- `POST {{base_url}}/v1/project-creation-sessions/{{pc_session_id}}/messages`

Body：

```json
{
  "prompt": "确认创建"
}
```

**断言**

- HTTP 200
- `data.project_id` 非空（项目已创建）
- `data.project_session_id` 非空（已关联长期会话）
- `data.session.status == "completed"`

**后置脚本**

```js
const res = pm.response.json();
pm.environment.set("project_id", res.data.project_id);
pm.environment.set("session_id", res.data.project_session_id);
```

### 2.6 验证项目已创建

**请求**

- `GET {{base_url}}/v1/projects/{{project_id}}`

**断言**

- HTTP 200
- `data.title` 和 `data.description` 非空

---

## 3. 用例 B：长期会话按任务切换生成设定集

> 说明：在已有项目上通过对话迭代设定。

### 3.1 生成世界观（worldview）

**请求**

- `POST {{base_url}}/v1/projects/{{project_id}}/sessions/{{session_id}}/messages`

Body：

```json
{
  "task": "worldview",
  "prompt": "请为本故事生成世界观设定：时间体系、城市结构、关键组织、记忆篡改技术的边界与代价。"
}
```

**断言**

- `data.artifact_snapshot.type == "worldview"`

### 3.2 生成角色设定（characters）

Body：

```json
{
  "task": "characters",
  "prompt": "请生成角色与关系：主角、反派、委托人、警方联系人。每个角色要有动机、弱点、与他人的关系。"
}
```

**断言**

- `data.artifact_snapshot.type == "characters"`

### 3.3 生成大纲（outline）

Body：

```json
{
  "task": "outline",
  "prompt": "请给出 3 卷结构，每卷 8 章，每章一句话大纲，并标注关键转折点。"
}
```

**断言**

- `data.artifact_snapshot.type == "outline"`

### 3.4 查看构件与版本列表

**构件列表**

- `GET {{base_url}}/v1/projects/{{project_id}}/artifacts`

**版本列表**

- `GET {{base_url}}/v1/projects/{{project_id}}/artifacts/{{artifact_id}}/versions`

### 3.5 回滚到指定版本

- `POST {{base_url}}/v1/projects/{{project_id}}/artifacts/{{artifact_id}}/rollback`

Body：

```json
{ "version_id": "替换为要回滚到的版本ID" }
```

---

## 4. 用例 C：Foundation 一期（Plan → Apply 落库）

### 4.1 预览生成 FoundationPlan

- `POST {{base_url}}/v1/projects/{{project_id}}/foundation/preview`

Body：

```json
{
  "prompt": "请为本小说生成设定集：世界观、角色、关系、3卷大纲（每卷6章）。"
}
```

**断言**

- HTTP 200
- `data.job_id` 与 `data.plan` 非空

### 4.2 Apply 落库

- `POST {{base_url}}/v1/projects/{{project_id}}/foundation/apply`

Body：

```json
{
  "job_id": "{{foundation_job_id}}"
}
```

**断言**

- HTTP 200
- `data.result.*_created/*_updated` 计数符合预期

### 4.3 验证落库结果

- `GET {{base_url}}/v1/projects/{{project_id}}/entities`
- `GET {{base_url}}/v1/projects/{{project_id}}/relations`
- `GET {{base_url}}/v1/projects/{{project_id}}/volumes`
- `GET {{base_url}}/v1/projects/{{project_id}}/chapters`

---

## 5. 常见问题与排查

| 状态码 | 原因                               | 解决方法                           |
| :----- | :--------------------------------- | :--------------------------------- |
| 401    | 未设置 Authorization 或 Token 过期 | 重新登录刷新                       |
| 403    | 账号角色无权限                     | 使用 admin 或 member 角色          |
| 429    | 触发租户 Token 日配额              | 调整租户配额或更换租户             |
| 500    | LLM 相关错误                       | 检查 provider/model 配置与 API Key |
