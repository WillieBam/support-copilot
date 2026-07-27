package db

import (
	"log"
	"time"

	"github.com/WillieBam/support_copilot/backend/types/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	realUserID     = uuid.MustParse("629dd75e-8677-4a06-91db-f3e379ea519f")
	superAdminID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	leadEngineerID = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	teamDevOpsID   = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	teamPlatformID = uuid.MustParse("44444444-4444-4444-4444-444444444444")
	engineerID1    = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

func InitDatabase(db *gorm.DB) {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Fatalf("Failed to create UUID extension: %v", err)
	}

	err := db.AutoMigrate(
		&models.User{},
		&models.Alert{},
		&models.Incident{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamIncident{},
		&models.IncidentStatusHistory{},
		&models.Conversation{},
		&models.Message{},
	)
	if err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}
	log.Println("Database migration completed successfully!")

	seedUsers(db)
	seedIncidents(db)
	seedTeamsAndMemberships(db)
	seedAlerts(db)
	seedTeamIncidents(db)
}

func seedUsers(db *gorm.DB) {
	defaultUsers := []models.User{
		{
			ID:          realUserID,
			FirebaseUID: "PrzOYbxjkQZU5pzmudAXXQrlf2G3",
			Email:       "meilin.22@1utar.my",
			DisplayName: "Meilin",
			Scope:       "engineer",
		},
		{
			ID:          superAdminID,
			FirebaseUID: "fb_superadmin_111",
			Email:       "superadmin@company.com",
			DisplayName: "System Boss",
			Scope:       "super_admin",
		},
		{
			ID:          leadEngineerID,
			FirebaseUID: "fb_lead_engineer_222",
			Email:       "lead.engineer@company.com",
			DisplayName: "Copper Lead",
			Scope:       "engineer",
		},
		{
			ID:          engineerID1,
			FirebaseUID: "fb_an_engineer_1",
			Email:       "an.engineer@company.com",
			DisplayName: "An En",
			Scope:       "engineer",
		},
	}

	for _, u := range defaultUsers {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "email"}},
			DoNothing: true,
		}).Create(&u).Error

		if err != nil {
			log.Printf("Warning: User seeding failed for %s: %v", u.Email, err)
		}
	}
	log.Println("User database seeding done!")
}

func seedIncidents(db *gorm.DB) {
	mockIncidentID := uuid.MustParse("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d")
	mockAlertID := uuid.MustParse("8c0ce31e-4c5d-4f3a-9e2a-1b0d7b3dcb6d")

	incidents := []models.Incident{
		{
			ID:        mockIncidentID,
			AlertID:   mockAlertID,
			UserID:    realUserID,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Details:   "Production high CPU load on payment-gateway-service",
			Status:    "OPEN",
		},
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&incidents).Error

	if err != nil {
		log.Printf("Warning: Incident seeding failed: %v", err)
	} else {
		log.Println("Root Incident database seeding done!")
	}
}

func seedTeamsAndMemberships(db *gorm.DB) {
	teams := []models.Team{
		{
			ID:        teamDevOpsID,
			TeamName:  "DevOps Rescue",
			CreatedAt: time.Now(),
		},
		{
			ID:        teamPlatformID,
			TeamName:  "Platform Bistro",
			CreatedAt: time.Now(),
		},
	}

	for _, t := range teams {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&t).Error
		if err != nil {
			log.Printf("Warning: Team seeding failed for %s: %v", t.TeamName, err)
		}
	}

	memberships := []models.TeamMember{
		{
			ID:     uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			TeamID: teamDevOpsID,
			UserID: realUserID,
			Role:   "owner",
		},
		{
			ID:     uuid.MustParse("66666666-6666-6666-6666-666666666666"),
			TeamID: teamDevOpsID,
			UserID: leadEngineerID,
			Role:   "member",
		},
		{
			ID:     uuid.MustParse("77777777-7777-7777-7777-777777777777"),
			TeamID: teamPlatformID,
			UserID: leadEngineerID,
			Role:   "owner",
		},
		{
			ID:     uuid.MustParse("88888888-8888-8888-8888-888888888888"),
			TeamID: teamPlatformID,
			UserID: realUserID,
			Role:   "member",
		},
	}

	for _, m := range memberships {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&m).Error

		if err != nil {
			log.Printf("Warning: TeamMember seeding failed: %v", err)
		}
	}
	log.Println("Team and Membership database seeding done!")
}

func seedAlerts(db *gorm.DB) {
	mockIncidentID := uuid.MustParse("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d")

	mockAlerts := []models.Alert{
		{
			ID:          uuid.New(),
			IncidentID:  mockIncidentID,
			ReceivedAt:  time.Now().Add(-15 * time.Minute),
			ServiceName: "payment-gateway-service",
			Severity:    "CRITICAL",
			Metrics:     `{"container.cpu.usage": 94.2, "runtime.go.mem_stats.total_alloc": 4825800, "error_rate": 0.06}`,
		},
		{
			ID:          uuid.New(),
			IncidentID:  mockIncidentID,
			ReceivedAt:  time.Now().Add(-5 * time.Minute),
			ServiceName: "authentication-service",
			Severity:    "WARNING",
			Metrics:     `{"trace.grpc.server.request.hits": 4500, "system.cpu.system": 78.1}`,
		},
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&mockAlerts).Error

	if err != nil {
		log.Printf("Warning: Alert seeding failed: %v", err)
	} else {
		log.Println("Alert database seeding done!")
	}
}

func seedTeamIncidents(db *gorm.DB) {
	mockIncidents := []models.TeamIncident{
		{
			ID:         uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			IncidentID: uuid.MustParse("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"),
			TeamID:     teamDevOpsID,
			AssignedBy: realUserID,
			Title:      "Payment Gateway High CPU Spike",
			Status:     "IN_PROGRESS",
			Details:    "CPU spike observed on payment gateway pod #3.",
			AssignedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:         uuid.MustParse("b2222222-2222-2222-2222-222222222222"),
			IncidentID: uuid.MustParse("9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"),
			TeamID:     teamPlatformID,
			AssignedBy: leadEngineerID,
			Title:      "Authentication Service Latency Degradation",
			Status:     "OPEN",
			Details:    "gRPC server response latency exceeded SLA threshold.",
			AssignedAt: time.Now().Add(-30 * time.Minute),
		},
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&mockIncidents).Error

	if err != nil {
		log.Printf("Warning: TeamIncident seeding failed: %v", err)
	} else {
		log.Println("TeamIncident database seeding done!")
	}
}
