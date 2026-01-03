// Package router 提供 HTTP 路由配置
package router

import (
	"z-novel-ai-api/internal/interfaces/http/handler"

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
		projects.GET("", projectHandler.ListProjects)
		projects.POST("", projectHandler.CreateProject)
		projects.GET("/:pid", projectHandler.GetProject)
		projects.PUT("/:pid", projectHandler.UpdateProject)
		projects.DELETE("/:pid", projectHandler.DeleteProject)

		// 项目设置
		projects.GET("/:pid/settings", projectHandler.GetProjectSettings)
		projects.PUT("/:pid/settings", projectHandler.UpdateProjectSettings)

		// 项目下的卷
		projects.GET("/:pid/volumes", volumeHandler.ListVolumes)
		projects.POST("/:pid/volumes", volumeHandler.CreateVolume)
		projects.POST("/:pid/volumes/reorder", volumeHandler.ReorderVolumes)

		// 项目下的章节
		projects.GET("/:pid/chapters", chapterHandler.ListChapters)
		projects.POST("/:pid/chapters", chapterHandler.CreateChapter)
		projects.POST("/:pid/chapters/generate", chapterHandler.GenerateChapter)

		// 项目下的实体
		projects.GET("/:pid/entities", entityHandler.ListEntities)
		projects.POST("/:pid/entities", entityHandler.CreateEntity)

		// 项目下的事件
		projects.GET("/:pid/events", eventHandler.ListEvents)
		projects.POST("/:pid/events", eventHandler.CreateEvent)

		// 项目下的关系
		projects.GET("/:pid/relations", relationHandler.ListRelations)
		projects.POST("/:pid/relations", relationHandler.CreateRelation)

		// 项目下的任务
		projects.GET("/:pid/jobs", jobHandler.ListProjectJobs)
	}

	// 事件管理
	events := v1.Group("/events")
	{
		events.GET("/:evid", eventHandler.GetEvent)
		events.PUT("/:evid", eventHandler.UpdateEvent)
		events.DELETE("/:evid", eventHandler.DeleteEvent)
	}

	// 关系管理
	relations := v1.Group("/relations")
	{
		relations.GET("/:rid", relationHandler.GetRelation)
		relations.PUT("/:rid", relationHandler.UpdateRelation)
		relations.DELETE("/:rid", relationHandler.DeleteRelation)
	}

	// 卷管理
	volumes := v1.Group("/volumes")
	{
		volumes.GET("/:vid", volumeHandler.GetVolume)
		volumes.PUT("/:vid", volumeHandler.UpdateVolume)
		volumes.DELETE("/:vid", volumeHandler.DeleteVolume)
	}

	// 章节管理
	chapters := v1.Group("/chapters")
	{
		chapters.GET("/:cid", chapterHandler.GetChapter)
		chapters.PUT("/:cid", chapterHandler.UpdateChapter)
		chapters.DELETE("/:cid", chapterHandler.DeleteChapter)
		chapters.GET("/:cid/stream", streamHandler.StreamChapter) // SSE
		chapters.POST("/:cid/regenerate", chapterHandler.RegenerateChapter)
	}

	// 实体管理
	entities := v1.Group("/entities")
	{
		entities.GET("/:eid", entityHandler.GetEntity)
		entities.PUT("/:eid", entityHandler.UpdateEntity)
		entities.DELETE("/:eid", entityHandler.DeleteEntity)
		entities.PUT("/:eid/state", entityHandler.UpdateEntityState)
		entities.GET("/:eid/relations", entityHandler.GetEntityRelations)
	}

	// 检索调试
	retrieval := v1.Group("/retrieval")
	{
		retrieval.POST("/search", retrievalHandler.Search)
		retrieval.POST("/debug", retrievalHandler.DebugRetrieval)
	}

	// 任务管理
	jobs := v1.Group("/jobs")
	{
		jobs.GET("/:jid", jobHandler.GetJob)
		jobs.DELETE("/:jid", jobHandler.CancelJob)
	}

	// 用户管理
	users := v1.Group("/users")
	{
		users.GET("/me", userHandler.GetMe)
		users.PUT("/me", userHandler.UpdateMe)
		users.GET("", userHandler.ListTenantUsers)
	}

	// 租户管理
	tenants := v1.Group("/tenants")
	{
		tenants.GET("/current", tenantHandler.GetCurrentTenant)
		tenants.PUT("/current", tenantHandler.UpdateCurrentTenant)
	}
}
