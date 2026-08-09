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

	db.Exec("ALTER TABLE IF EXISTS team_incidents DROP COLUMN IF EXISTS incident_id")
	db.Exec("UPDATE alerts SET incident_id = NULL WHERE incident_id IS NOT NULL AND incident_id NOT IN (SELECT id FROM team_incidents)")
	db.Exec("UPDATE runbooks SET incident_id = 'a1111111-1111-1111-1111-111111111111' WHERE incident_id NOT IN (SELECT id FROM team_incidents)")
	db.Exec("UPDATE runbook_logs SET incident_id = 'a1111111-1111-1111-1111-111111111111' WHERE incident_id NOT IN (SELECT id FROM team_incidents)")
	db.Exec("UPDATE conversations SET team_incident_id = NULL WHERE team_incident_id IS NOT NULL AND team_incident_id NOT IN (SELECT id FROM team_incidents)")
	db.Exec("UPDATE messages SET parent_message_id = NULL WHERE parent_message_id IS NOT NULL AND parent_message_id NOT IN (SELECT id FROM messages)")

	err := db.AutoMigrate(
		&models.User{},
		&models.Team{},
		&models.TeamMember{},
		&models.TeamIncident{},
		&models.IncidentStatusHistory{},
		&models.Instruction{},
		&models.InstructionLog{},
		&models.Runbook{},
		&models.RunbookLog{},
		&models.Alert{},
		&models.Conversation{},
		&models.Message{},
	)
	if err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}
	log.Println("Database migration completed successfully!")

	seedUsers(db)
	seedTeamsAndMemberships(db)
	seedTeamIncidents(db)
	seedAlerts(db)
	seedRunbooks(db)
	seedRunbookLogs(db)
	seedInstructions(db)
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
	mockIncidentID := uuid.MustParse("a1111111-1111-1111-1111-111111111111")

	mockAlerts := []models.Alert{
		{
			ID:              uuid.New(),
			IncidentID:      &mockIncidentID,
			ReceivedAt:      time.Now().Add(-15 * time.Minute),
			ResourceInfo:    `{"service":"payment-gateway-service"}`,
			AlertInfo:       `{"severity":"CRITICAL"}`,
			Metrics:         `{"container.cpu.usage": 94.2, "runtime.go.mem_stats.total_alloc": 4825800, "error_rate": 0.06}`,
			BusinessContext: "{}",
			Metadata:        "{}",
		},
		{
			ID:              uuid.New(),
			IncidentID:      &mockIncidentID,
			ReceivedAt:      time.Now().Add(-5 * time.Minute),
			ResourceInfo:    `{"service":"authentication-service"}`,
			AlertInfo:       `{"severity":"WARNING"}`,
			Metrics:         `{"trace.grpc.server.request.hits": 4500, "system.cpu.system": 78.1}`,
			BusinessContext: "{}",
			Metadata:        "{}",
		},
		{
			ID:              uuid.New(),
			IncidentID:      nil,
			ReceivedAt:      time.Now().Add(-2 * time.Minute),
			ResourceInfo:    `{"service":"report-upload-service"}`,
			AlertInfo:       `{"severity":"CRITICAL"}`,
			Metrics:         `{"cpu_usage": 96.8, "response_latency": 450.0, "error_rate": 0.08}`,
			BusinessContext: "{}",
			Metadata:        "{}",
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
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Payment Gateway High CPU Spike",
			Status:     "IN_PROGRESS",
			Details:    "CPU spike observed on payment gateway pod #3.",
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			AssignedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:         uuid.MustParse("a2222222-2222-2222-2222-222222222222"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Redis Session Cache Out of Memory",
			Status:     "OPEN",
			Details:    "Eviction policy maxmemory exceeded on cache-cluster-01.",
			CreatedAt:  time.Now().Add(-45 * time.Minute),
			AssignedAt: time.Now().Add(-45 * time.Minute),
		},
		{
			ID:         uuid.MustParse("a3333333-3333-3333-3333-333333333333"),
			TeamID:     teamDevOpsID,
			CreatedBy:  leadEngineerID,
			Title:      "Cart Service DB Connection Pool Exhausted",
			Status:     "IN_PROGRESS",
			Details:    "Connection pool leak detected in cart-service v2.4.",
			CreatedAt:  time.Now().Add(-3 * time.Hour),
			AssignedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			ID:         uuid.MustParse("a4444444-4444-4444-4444-444444444444"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Kafka Consumer Lag Spike on Orders Topic",
			Status:     "OPEN",
			Details:    "Lag count exceeded 50,000 unhandled messages.",
			CreatedAt:  time.Now().Add(-15 * time.Minute),
			AssignedAt: time.Now().Add(-15 * time.Minute),
		},
		{
			ID:         uuid.MustParse("a5555555-5555-5555-5555-555555555555"),
			TeamID:     teamDevOpsID,
			CreatedBy:  engineerID1,
			Title:      "Search Index Synchronization Out of Sync",
			Status:     "OPEN",
			Details:    "Elasticsearch cluster node #2 rebalancing failure.",
			CreatedAt:  time.Now().Add(-10 * time.Minute),
			AssignedAt: time.Now().Add(-10 * time.Minute),
		},
		{
			ID:         uuid.MustParse("a6666666-6666-6666-6666-666666666666"),
			TeamID:     teamDevOpsID,
			CreatedBy:  superAdminID,
			Title:      "Kubernetes Node Network Partition",
			Status:     "RESOLVED",
			Details:    "Node replaced and network overlay routing rules restored.",
			CreatedAt:  time.Now().Add(-12 * time.Hour),
			AssignedAt: time.Now().Add(-12 * time.Hour),
			ResolvedAt: timePtr(time.Now().Add(-5 * time.Hour)),
		},
		{
			ID:         uuid.MustParse("a7777777-7777-7777-7777-777777777777"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Ingress Controller SSL Certificate Expiry",
			Status:     "RESOLVED",
			Details:    "Cert-manager automatically renewed wildcard TLS certificate.",
			CreatedAt:  time.Now().Add(-24 * time.Hour),
			AssignedAt: time.Now().Add(-24 * time.Hour),
			ResolvedAt: timePtr(time.Now().Add(-23 * time.Hour)),
		},
		{
			ID:         uuid.MustParse("a8888888-8888-8888-8888-888888888888"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Database Deadlock in User Service",
			Status:     "RESOLVED",
			Details:    "Optimized transaction order to eliminate deadlocks.",
			CreatedAt:  time.Now().Add(-18 * time.Hour),
			AssignedAt: time.Now().Add(-18 * time.Hour),
			ResolvedAt: timePtr(time.Now().Add(-17 * time.Hour)),
		},
		{
			ID:         uuid.MustParse("a9999999-9999-9999-9999-999999999999"),
			TeamID:     teamDevOpsID,
			CreatedBy:  leadEngineerID,
			Title:      "Disk Space Exhaustion on Log Server",
			Status:     "RESOLVED",
			Details:    "Cleared archived logs and updated retention policy.",
			CreatedAt:  time.Now().Add(-8 * time.Hour),
			AssignedAt: time.Now().Add(-8 * time.Hour),
			ResolvedAt: timePtr(time.Now().Add(-6 * time.Hour)),
		},
		{
			ID:         uuid.MustParse("b2222222-2222-2222-2222-222222222222"),
			TeamID:     teamPlatformID,
			CreatedBy:  leadEngineerID,
			Title:      "Authentication Service Latency Degradation",
			Status:     "IN_PROGRESS",
			Details:    "gRPC server response latency exceeded SLA threshold.",
			CreatedAt:  time.Now().Add(-30 * time.Minute),
			AssignedAt: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:         uuid.MustParse("b3333333-3333-3333-3333-333333333333"),
			TeamID:     teamPlatformID,
			CreatedBy:  realUserID,
			Title:      "GraphQL Gateway Schema Stitching Failure",
			Status:     "RESOLVED",
			Details:    "Rolled back breaking change in catalog microservice deployment.",
			CreatedAt:  time.Now().Add(-6 * time.Hour),
			AssignedAt: time.Now().Add(-6 * time.Hour),
			ResolvedAt: timePtr(time.Now().Add(-5 * time.Hour)),
		},
	}

	for _, inc := range mockIncidents {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&inc).Error
		if err != nil {
			log.Printf("Warning: TeamIncident seeding failed for %s: %v", inc.Title, err)
		}
	}
	log.Println("TeamIncident database seeding done!")

	seedIncidentStatusHistory(db)
}

func seedIncidentStatusHistory(db *gorm.DB) {
	mockHistories := []models.IncidentStatusHistory{
		{
			ID:             uuid.MustParse("c1111111-1111-1111-1111-111111111111"),
			TeamIncidentID: uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			UpdatedBy:      realUserID,
			Title:          "Payment Gateway High CPU Spike",
			PreviousStatus: "",
			NewStatus:      "OPEN",
			Details:        "Incident initialized from critical alert.",
			UpdatedAt:      time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             uuid.MustParse("c2222222-2222-2222-2222-222222222222"),
			TeamIncidentID: uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			UpdatedBy:      realUserID,
			Title:          "Payment Gateway High CPU Spike",
			PreviousStatus: "OPEN",
			NewStatus:      "IN_PROGRESS",
			Details:        "Assigned on-call engineer Meilin. Scaling pod replicas from 3 to 6.",
			UpdatedAt:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:             uuid.MustParse("c3333333-3333-3333-3333-333333333333"),
			TeamIncidentID: uuid.MustParse("a3333333-3333-3333-3333-333333333333"),
			UpdatedBy:      leadEngineerID,
			Title:          "Cart Service DB Connection Pool Exhausted",
			PreviousStatus: "",
			NewStatus:      "OPEN",
			Details:        "High connection count observed on primary DB node.",
			UpdatedAt:      time.Now().Add(-3 * time.Hour),
		},
		{
			ID:             uuid.MustParse("c4444444-4444-4444-4444-444444444444"),
			TeamIncidentID: uuid.MustParse("a3333333-3333-3333-3333-333333333333"),
			UpdatedBy:      leadEngineerID,
			Title:          "Cart Service DB Connection Pool Exhausted",
			PreviousStatus: "OPEN",
			NewStatus:      "IN_PROGRESS",
			Details:        "Investigating connection leak in cart-service v2.4 pool driver.",
			UpdatedAt:      time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             uuid.MustParse("c5555555-5555-5555-5555-555555555555"),
			TeamIncidentID: uuid.MustParse("a6666666-6666-6666-6666-666666666666"),
			UpdatedBy:      superAdminID,
			Title:          "Kubernetes Node Network Partition",
			PreviousStatus: "OPEN",
			NewStatus:      "IN_PROGRESS",
			Details:        "Replacing faulty node worker-04.",
			UpdatedAt:      time.Now().Add(-10 * time.Hour),
		},
		{
			ID:             uuid.MustParse("c6666666-6666-6666-6666-666666666666"),
			TeamIncidentID: uuid.MustParse("a6666666-6666-6666-6666-666666666666"),
			UpdatedBy:      superAdminID,
			Title:          "Kubernetes Node Network Partition",
			PreviousStatus: "IN_PROGRESS",
			NewStatus:      "RESOLVED",
			Details:        "Node replaced and network overlay routing rules restored.",
			UpdatedAt:      time.Now().Add(-5 * time.Hour),
		},
	}

	for _, h := range mockHistories {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&h).Error
		if err != nil {
			log.Printf("Warning: IncidentStatusHistory seeding failed: %v", err)
		}
	}
	log.Println("IncidentStatusHistory database seeding done!")
}

func seedInstructions(db *gorm.DB) {
	mockDevOpsInstructionID := uuid.MustParse("d1111111-1111-1111-1111-111111111111")

	instruction := models.Instruction{
		ID:                 mockDevOpsInstructionID,
		CreatedBy:          realUserID,
		TeamID:             teamDevOpsID,
		InstructionDetails: "Always prioritize high CPU and memory leak alerts for payment-gateway services. When responding, provide concise technical diagnosis steps and recommended kubectl / container remediation commands.",
		CreatedAt:          time.Now().Add(-24 * time.Hour),
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&instruction).Error
	if err != nil {
		log.Printf("Warning: Instruction seeding failed: %v", err)
	}

	history := models.InstructionLog{
		ID:               uuid.MustParse("e1111111-1111-1111-1111-111111111111"),
		InstructionID:    mockDevOpsInstructionID,
		UpdatedBy:        realUserID,
		OlderInstruction: "Initial instructions: Assist DevOps engineers with incident analysis.",
		Version:          1,
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	err = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&history).Error
	if err != nil {
		log.Printf("Warning: InstructionLog seeding failed: %v", err)
	} else {
		log.Println("Instruction database seeding done!")
	}
}

func seedRunbooks(db *gorm.DB) {
	mockRunbooks := []models.Runbook{
		{
			ID:         uuid.MustParse("f1111111-1111-1111-1111-111111111111"),
			IncidentID: uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "Payment Gateway CPU Spike — Scale & Throttle",
			Status:     "active",
			Content: "## Root Cause\n" +
				"Pod CPU throttling due to burst limit misconfiguration on payment-gateway-service.\n\n" +
				"## Diagnostic Steps\n" +
				"1. Check HPA thresholds: `kubectl get hpa -n production`\n" +
				"2. Inspect pod CPU limits: `kubectl describe pod <pod-name> -n production`\n" +
				"3. Review recent deployments: `kubectl rollout history deploy/payment-gateway`\n\n" +
				"## Resolution\n" +
				"1. Scale replicas immediately: `kubectl scale deploy payment-gateway --replicas=6 -n production`\n" +
				"2. Adjust HPA `maxReplicas` to 10 and `targetCPUUtilizationPercentage` to 60\n" +
				"3. Apply circuit breaker config to prevent downstream cascade\n\n" +
				"## Prevention\n" +
				"Set pod CPU requests/limits correctly and enable auto-scaling alerts at 70% threshold.",
		},
		{
			ID:         uuid.MustParse("f2222222-2222-2222-2222-222222222222"),
			IncidentID: uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			TeamID:     teamDevOpsID,
			CreatedBy:  leadEngineerID,
			Title:      "Old: Manual Pod Restart (Deprecated)",
			Status:     "deprecated",
			Content:    "Manually restart pods via `kubectl rollout restart deploy/payment-gateway`. Deprecated in favour of auto-scaling runbook f1111111.",
		},
		{
			ID:         uuid.MustParse("f3333333-3333-3333-3333-333333333333"),
			IncidentID: uuid.MustParse("a8888888-8888-8888-8888-888888888888"),
			TeamID:     teamDevOpsID,
			CreatedBy:  realUserID,
			Title:      "PostgreSQL Connection Pool Exhaustion & Recovery",
			Status:     "active",
			Content: "## Symptom\n" +
				"Applications logging `pg_stat_activity: FATAL: remaining connection slots reserved for non-replication superuser connections`.\n\n" +
				"## Diagnostic Steps\n" +
				"1. Query active pool usage: `SELECT count(*), state FROM pg_stat_activity GROUP BY state;`\n" +
				"2. Identify idle connections: `SELECT pid, now() - query_start AS duration, query FROM pg_stat_activity WHERE state = 'idle in transaction' ORDER BY duration DESC;`\n\n" +
				"## Remediation\n" +
				"1. Terminate runaway idle connections: `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle in transaction' AND now() - query_start > interval '5 minutes';`\n" +
				"2. Scale up PgBouncer pooler replicas or adjust `max_client_conn` setting.\n" +
				"3. Restart application deployment if connection leak persists.",
		},
		{
			ID:         uuid.MustParse("f4444444-4444-4444-4444-444444444444"),
			IncidentID: uuid.MustParse("a2222222-2222-2222-2222-222222222222"),
			TeamID:     teamDevOpsID,
			CreatedBy:  engineerID1,
			Title:      "Redis Cache Eviction Surge — Memory Remediation",
			Status:     "active",
			Content: "## Trigger\n" +
				"Alert `RedisMemoryHigh` (>90% maxmemory) with spike in `evicted_keys` per second.\n\n" +
				"## Investigation\n" +
				"1. Connect to Redis CLI: `redis-cli -h $REDIS_HOST info memory`\n" +
				"2. Check current eviction policy: `redis-cli -h $REDIS_HOST config get maxmemory-policy`\n" +
				"3. Check big keys: `redis-cli -h $REDIS_HOST --bigkeys`\n\n" +
				"## Resolution Steps\n" +
				"1. Flush volatile non-critical session keys if TTL is set: `redis-cli EVAL \"...\"`\n" +
				"2. Temporarily increase memory limit via cloud console or Helm chart value.\n" +
				"3. Update eviction policy to `volatile-lru` if currently `noeviction`.",
		},
		{
			ID:         uuid.MustParse("f5555555-5555-5555-5555-555555555555"),
			IncidentID: uuid.MustParse("b2222222-2222-2222-2222-222222222222"),
			TeamID:     teamPlatformID,
			CreatedBy:  leadEngineerID,
			Title:      "API Gateway Ingress Rate Limiting & Circuit Breaking",
			Status:     "active",
			Content: "## Overview\n" +
				"Runbook for platform team managing Envoy/Kong API Gateway rate limits during traffic spikes.\n\n" +
				"## Diagnostics\n" +
				"1. Monitor 429 Too Many Requests response rates across ingress routes.\n" +
				"2. Inspect active rate limiter token bucket status via Prometheus metrics.\n\n" +
				"## Resolution\n" +
				"1. Temporarily increase burst limit threshold in Envoy filter config.\n" +
				"2. Enable IP-based throttling for abusive client user-agents.",
		},
	}

	for _, rb := range mockRunbooks {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&rb).Error
		if err != nil {
			log.Printf("Warning: Runbook seeding failed for %s: %v", rb.Title, err)
		}
	}
	log.Println("Runbook database seeding done!")
}

func seedRunbookLogs(db *gorm.DB) {
	mockLogs := []models.RunbookLog{
		{
			ID:           uuid.MustParse("f6666666-6666-6666-6666-666666666666"),
			RunbookID:    uuid.MustParse("f1111111-1111-1111-1111-111111111111"),
			IncidentID:   uuid.MustParse("a1111111-1111-1111-1111-111111111111"),
			TeamID:       teamDevOpsID,
			UpdatedBy:    realUserID,
			OlderTitle:   "Payment Gateway CPU Spike — Initial Version",
			OlderContent: "Initial draft: restart payment gateway pod when CPU hits 90%.",
			Version:      1,
			UpdatedAt:    time.Now().Add(-12 * time.Hour),
		},
	}

	for _, l := range mockLogs {
		err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoNothing: true,
		}).Create(&l).Error
		if err != nil {
			log.Printf("Warning: RunbookLog seeding failed: %v", err)
		}
	}
	log.Println("RunbookLog database seeding done!")
}

func timePtr(t time.Time) *time.Time {
	return &t
}

