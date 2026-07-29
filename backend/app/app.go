package app

import (
	"fmt"
	"log"

	"github.com/WillieBam/support_copilot/backend/app/config"
	"github.com/WillieBam/support_copilot/backend/internal/interfaces"
	firebaseRepo "github.com/WillieBam/support_copilot/backend/internal/repository/firebase"
	llm "github.com/WillieBam/support_copilot/backend/internal/repository/llm"
	mcp "github.com/WillieBam/support_copilot/backend/internal/repository/mcp"
	postgresRepo "github.com/WillieBam/support_copilot/backend/internal/repository/postgres"
	"github.com/WillieBam/support_copilot/backend/internal/service"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type App struct {
	Repository  *AppRepository
	Service     interfaces.IAppService
	AuthService interfaces.IAuthService
	TeamService interfaces.ITeamService
}

func NewApp() *App {
	cfg := config.Get()

	// Open DB connection
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
	)
	gormDB, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	userRepo := postgresRepo.NewUserRepository(gormDB)
	alertRepo := postgresRepo.NewAlertRepository(gormDB)
	teamRepo := postgresRepo.NewTeamRepository(gormDB)
	convRepo := postgresRepo.NewConversationRepository(gormDB)
	llmClient := llm.NewOllamaClient(cfg)
	mcpOneClient := mcp.NewMcpOneClient(cfg)

	appRepository := NewAppRepository(llmClient, userRepo, alertRepo, teamRepo, convRepo)

	firebaseRepository, err := firebaseRepo.NewFirebaseRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase Repository: %v", err)
	}

	// initialize authentication service
	authService := service.New(service.AuthServiceParam{
		UserRepo:     appRepository.User,
		FirebaseRepo: firebaseRepository,
	})

	appService := service.NewAppService(appRepository.Alert, appRepository.LLM, mcpOneClient, convRepo, appRepository.Team)
	teamService := service.NewTeamService(appRepository.Team)

	return &App{
		Repository:  appRepository,
		Service:     appService,
		AuthService: authService,
		TeamService: teamService,
	}
}
