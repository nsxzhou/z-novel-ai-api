package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"z-novel-ai-api/internal/config"
	"z-novel-ai-api/internal/domain/entity"
	"z-novel-ai-api/internal/wire"
)

func main() {
	_ = godotenv.Load()

	fmt.Println("Starting system bootstrap...")

	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	// 2. 初始化数据层（仅 PostgreSQL）
	dataLayer, cleanup, err := wire.InitializePostgresOnly(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize data layer: %v", err)
	}
	defer cleanup()

	// 3. 创建默认租户
	defaultTenantSlug := "default-tenant"
	exists, err := dataLayer.TenantRepo.ExistsBySlug(ctx, defaultTenantSlug)
	if err != nil {
		log.Fatalf("failed to check tenant existence: %v", err)
	}

	var tenantID string
	if !exists {
		fmt.Printf("Creating default tenant: %s...\n", defaultTenantSlug)
		tenant := entity.NewTenant("Default Tenant", defaultTenantSlug)
		if err := dataLayer.TenantRepo.Create(ctx, tenant); err != nil {
			log.Fatalf("failed to create default tenant: %v", err)
		}
		tenantID = tenant.ID
		fmt.Printf("Default tenant created with ID: %s\n", tenantID)
	} else {
		tenant, err := dataLayer.TenantRepo.GetBySlug(ctx, defaultTenantSlug)
		if err != nil {
			log.Fatalf("failed to get existing tenant: %v", err)
		}
		tenantID = tenant.ID
		fmt.Printf("Default tenant already exists with ID: %s\n", tenantID)
	}

	// 4. 创建首个管理员
	adminEmail := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@nsxzhou.fun"
	}
	adminPassword := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123" // 生产环境请务必通过环境变量设置
	}

	userExists, err := dataLayer.UserRepo.ExistsByEmail(ctx, tenantID, adminEmail)
	if err != nil {
		log.Fatalf("failed to check admin existence: %v", err)
	}

	if !userExists {
		fmt.Printf("Creating admin user: %s...\n", adminEmail)
		admin := entity.NewUser(tenantID, adminEmail, "System Admin")
		admin.Role = entity.UserRoleAdmin
		if err := admin.SetPassword(adminPassword); err != nil {
			log.Fatalf("failed to hash admin password: %v", err)
		}

		if err := dataLayer.UserRepo.Create(ctx, admin); err != nil {
			log.Fatalf("failed to create admin user: %v", err)
		}
		fmt.Printf("Admin user created successfully.\n")
	} else {
		fmt.Printf("Admin user %s already exists.\n", adminEmail)
	}

	fmt.Println("Bootstrap completed successfully.")
}
