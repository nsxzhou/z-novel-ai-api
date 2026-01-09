// Package router 提供 HTTP 路由配置
package router

import (
	"z-novel-ai-api/internal/interfaces/http/handler"
	"z-novel-ai-api/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterV1Routes 注册 v1 版本路由
func RegisterV1Routes(
	v1 *gin.RouterGroup,
	authHandler *handler.AuthHandler,
	projectHandler *handler.ProjectHandler,
	volumeHandler *handler.VolumeHandler,
	chapterHandler *handler.ChapterHandler,
	entityHandler *handler.EntityHandler,
	foundationHandler *handler.FoundationHandler,
	conversationHandler *handler.ConversationHandler,
	projectCreationHandler *handler.ProjectCreationHandler,
	artifactHandler *handler.ArtifactHandler,
	jobHandler *handler.JobHandler,
	retrievalHandler *handler.RetrievalHandler,
	streamHandler *handler.StreamHandler,
	userHandler *handler.UserHandler,
	tenantHandler *handler.TenantHandler,
	eventHandler *handler.EventHandler,
	relationHandler *handler.RelationHandler,
) {
	// 认证管理
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/logout", authHandler.Logout)
	}

	// 项目管理
	projects := v1.Group("/projects")
	{
		// 读操作（显式校验 PermProjectRead 权限）
		projects.GET("", middleware.RequirePermission(middleware.PermProjectRead), projectHandler.ListProjects)
		projects.GET("/:pid", middleware.RequirePermission(middleware.PermProjectRead), projectHandler.GetProject)
		projects.GET("/:pid/settings", middleware.RequirePermission(middleware.PermProjectRead), projectHandler.GetProjectSettings)
		projects.GET("/:pid/volumes", middleware.RequirePermission(middleware.PermProjectRead), volumeHandler.ListVolumes)
		projects.GET("/:pid/chapters", middleware.RequirePermission(middleware.PermProjectRead), chapterHandler.ListChapters)
		projects.GET("/:pid/entities", middleware.RequirePermission(middleware.PermProjectRead), entityHandler.ListEntities)
		projects.GET("/:pid/events", middleware.RequirePermission(middleware.PermProjectRead), eventHandler.ListEvents)
		projects.GET("/:pid/relations", middleware.RequirePermission(middleware.PermProjectRead), relationHandler.ListRelations)
		projects.GET("/:pid/jobs", middleware.RequirePermission(middleware.PermProjectRead), jobHandler.ListProjectJobs)

		// 写操作（需要 project:write 权限）
		projects.POST("", middleware.RequirePermission(middleware.PermProjectWrite), projectHandler.CreateProject)
		projects.PUT("/:pid", middleware.RequirePermission(middleware.PermProjectWrite), projectHandler.UpdateProject)
		projects.DELETE("/:pid", middleware.RequirePermission(middleware.PermProjectWrite), projectHandler.DeleteProject)
		projects.PUT("/:pid/settings", middleware.RequirePermission(middleware.PermProjectWrite), projectHandler.UpdateProjectSettings)

		// 卷写操作
		projects.POST("/:pid/volumes", middleware.RequirePermission(middleware.PermProjectWrite), volumeHandler.CreateVolume)
		projects.POST("/:pid/volumes/reorder", middleware.RequirePermission(middleware.PermProjectWrite), volumeHandler.ReorderVolumes)

		// 章节写操作
		projects.POST("/:pid/chapters", middleware.RequirePermission(middleware.PermProjectWrite), chapterHandler.CreateChapter)

		// 章节生成（需要 chapter:generate 权限）
		projects.POST("/:pid/chapters/generate", middleware.RequirePermission(middleware.PermChapterGenerate), chapterHandler.GenerateChapter)

		// 设定集生成（一期：复用 chapter:generate；落库 apply 需要 project:write）
		projects.POST("/:pid/foundation/preview", middleware.RequirePermission(middleware.PermChapterGenerate), foundationHandler.PreviewFoundation)
		projects.GET("/:pid/foundation/stream", middleware.RequirePermission(middleware.PermChapterGenerate), foundationHandler.StreamFoundation)  // SSE (GET)
		projects.POST("/:pid/foundation/stream", middleware.RequirePermission(middleware.PermChapterGenerate), foundationHandler.StreamFoundation) // SSE (POST)
		projects.POST("/:pid/foundation/generate", middleware.RequirePermission(middleware.PermChapterGenerate), foundationHandler.GenerateFoundation)
		projects.POST("/:pid/foundation/apply", middleware.RequirePermission(middleware.PermProjectWrite), foundationHandler.ApplyFoundation)

		// 长期会话（按任务切换生成构件版本；写操作需要 project:write）
		projects.POST("/:pid/sessions", middleware.RequirePermission(middleware.PermProjectWrite), conversationHandler.CreateSession)
		projects.GET("/:pid/sessions/:sid", middleware.RequirePermission(middleware.PermProjectRead), conversationHandler.GetSession)
		projects.GET("/:pid/sessions/:sid/turns", middleware.RequirePermission(middleware.PermProjectRead), conversationHandler.ListTurns)
		projects.POST("/:pid/sessions/:sid/messages", middleware.RequirePermission(middleware.PermProjectWrite), conversationHandler.SendMessage)

		// 构件版本（读：project:read；回滚：project:write）
		projects.GET("/:pid/artifacts", middleware.RequirePermission(middleware.PermProjectRead), artifactHandler.ListArtifacts)
		projects.GET("/:pid/artifacts/:aid/versions", middleware.RequirePermission(middleware.PermProjectRead), artifactHandler.ListVersions)
		projects.GET("/:pid/artifacts/:aid/branches", middleware.RequirePermission(middleware.PermProjectRead), artifactHandler.ListBranches)
		projects.GET("/:pid/artifacts/:aid/compare", middleware.RequirePermission(middleware.PermProjectRead), artifactHandler.CompareVersions)
		projects.POST("/:pid/artifacts/:aid/rollback", middleware.RequirePermission(middleware.PermProjectWrite), artifactHandler.Rollback)

		// 实体写操作
		projects.POST("/:pid/entities", middleware.RequirePermission(middleware.PermProjectWrite), entityHandler.CreateEntity)

		// 事件写操作
		projects.POST("/:pid/events", middleware.RequirePermission(middleware.PermProjectWrite), eventHandler.CreateEvent)

		// 关系写操作
		projects.POST("/:pid/relations", middleware.RequirePermission(middleware.PermProjectWrite), relationHandler.CreateRelation)
	}

	// 对话创建项目（Project 未存在时的会话）
	projectCreation := v1.Group("/project-creation-sessions")
	{
		projectCreation.POST("", middleware.RequirePermission(middleware.PermProjectWrite), projectCreationHandler.CreateSession)
		projectCreation.POST("/:sid/messages", middleware.RequirePermission(middleware.PermProjectWrite), projectCreationHandler.SendMessage)
		projectCreation.GET("/:sid", middleware.RequirePermission(middleware.PermProjectRead), projectCreationHandler.GetSession)
		projectCreation.GET("/:sid/turns", middleware.RequirePermission(middleware.PermProjectRead), projectCreationHandler.ListTurns)
	}

	// 事件管理
	events := v1.Group("/events")
	{
		events.GET("/:evid", middleware.RequirePermission(middleware.PermProjectRead), eventHandler.GetEvent)
		events.PUT("/:evid", middleware.RequirePermission(middleware.PermProjectWrite), eventHandler.UpdateEvent)
		events.DELETE("/:evid", middleware.RequirePermission(middleware.PermProjectWrite), eventHandler.DeleteEvent)
	}

	// 关系管理
	relations := v1.Group("/relations")
	{
		relations.GET("/:rid", middleware.RequirePermission(middleware.PermProjectRead), relationHandler.GetRelation)
		relations.PUT("/:rid", middleware.RequirePermission(middleware.PermProjectWrite), relationHandler.UpdateRelation)
		relations.DELETE("/:rid", middleware.RequirePermission(middleware.PermProjectWrite), relationHandler.DeleteRelation)
	}

	// 卷管理
	volumes := v1.Group("/volumes")
	{
		volumes.GET("/:vid", middleware.RequirePermission(middleware.PermProjectRead), volumeHandler.GetVolume)
		volumes.PUT("/:vid", middleware.RequirePermission(middleware.PermProjectWrite), volumeHandler.UpdateVolume)
		volumes.DELETE("/:vid", middleware.RequirePermission(middleware.PermProjectWrite), volumeHandler.DeleteVolume)
	}

	// 章节管理
	chapters := v1.Group("/chapters")
	{
		chapters.GET("/:cid", middleware.RequirePermission(middleware.PermProjectRead), chapterHandler.GetChapter)
		chapters.GET("/:cid/stream", middleware.RequirePermission(middleware.PermChapterGenerate), streamHandler.StreamChapter) // SSE
		chapters.PUT("/:cid", middleware.RequirePermission(middleware.PermProjectWrite), chapterHandler.UpdateChapter)
		chapters.DELETE("/:cid", middleware.RequirePermission(middleware.PermProjectWrite), chapterHandler.DeleteChapter)
		chapters.POST("/:cid/regenerate", middleware.RequirePermission(middleware.PermChapterGenerate), chapterHandler.RegenerateChapter)
	}

	// 实体管理
	entities := v1.Group("/entities")
	{
		entities.GET("/:eid", middleware.RequirePermission(middleware.PermProjectRead), entityHandler.GetEntity)
		entities.GET("/:eid/relations", middleware.RequirePermission(middleware.PermProjectRead), entityHandler.GetEntityRelations)
		entities.PUT("/:eid", middleware.RequirePermission(middleware.PermProjectWrite), entityHandler.UpdateEntity)
		entities.DELETE("/:eid", middleware.RequirePermission(middleware.PermProjectWrite), entityHandler.DeleteEntity)
		entities.PUT("/:eid/state", middleware.RequirePermission(middleware.PermProjectWrite), entityHandler.UpdateEntityState)
	}

	// 检索调试
	retrieval := v1.Group("/retrieval")
	{
		retrieval.POST("/search", middleware.RequirePermission(middleware.PermProjectRead), retrievalHandler.Search)
		retrieval.POST("/debug", middleware.RequirePermission(middleware.PermProjectRead), retrievalHandler.DebugRetrieval)
	}

	// 任务管理
	jobs := v1.Group("/jobs")
	{
		jobs.GET("/:jid", middleware.RequirePermission(middleware.PermProjectRead), jobHandler.GetJob)
		jobs.DELETE("/:jid", middleware.RequirePermission(middleware.PermProjectWrite), jobHandler.CancelJob)
	}

	// 用户管理
	users := v1.Group("/users")
	{
		// 个人操作（所有已认证用户可访问）
		users.GET("/me", userHandler.GetMe)
		users.PUT("/me", userHandler.UpdateMe)

		// 租户内用户列表（所有已认证用户可访问）
		users.GET("", userHandler.ListTenantUsers)

		// 管理操作（仅 admin 可访问）
		users.PUT("/:id/role", middleware.RequireAdmin(), userHandler.UpdateUserRole)
		users.DELETE("/:id", middleware.RequireAdmin(), userHandler.DeleteUser)
	}

	// 租户管理
	tenants := v1.Group("/tenants")
	{
		// 当前租户操作（所有已认证用户可访问当前租户信息）
		tenants.GET("/current", tenantHandler.GetCurrentTenant)

		// 管理操作（仅 admin 可访问）
		tenants.PUT("/current", middleware.RequireAdmin(), tenantHandler.UpdateCurrentTenant)
		tenants.GET("", middleware.RequireAdmin(), tenantHandler.ListTenants)
		tenants.POST("", middleware.RequireAdmin(), tenantHandler.CreateTenant)
	}
}
