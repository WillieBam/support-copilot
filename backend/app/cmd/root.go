package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/WillieBam/support_copilot/backend/app/config"
	seeds "github.com/WillieBam/support_copilot/db"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var rootCmd = &cobra.Command{
	Use:   "support-copilot",
	Short: "Support copilot",
	Long:  "Support Copilot",
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database auto-migration and seeding",
	Long:  "Run database auto-migration and seeding",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
			cfg.Database.Host,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Name,
			cfg.Database.Port,
		)

		gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		seeds.InitDatabase(gormDB)

	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
