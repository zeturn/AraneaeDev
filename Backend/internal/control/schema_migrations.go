package control

import (
	"araneae-go/internal/common"

	"gorm.io/gorm"
)

func reconcileSchema(db *gorm.DB) error {
	migrator := db.Migrator()
	if migrator.HasIndex(&common.RSSSubscription{}, "idx_rss_subscriptions_url") {
		if err := migrator.DropIndex(&common.RSSSubscription{}, "idx_rss_subscriptions_url"); err != nil {
			return err
		}
	}
	if !migrator.HasIndex(&common.RSSSubscription{}, "idx_rss_workplace_url") {
		if err := migrator.CreateIndex(&common.RSSSubscription{}, "idx_rss_workplace_url"); err != nil {
			return err
		}
	}
	if err := db.Model(&common.Schedule{}).
		Where("trigger_type = '' OR trigger_type IS NULL").
		Update("trigger_type", "api").Error; err != nil {
		return err
	}
	if migrator.HasColumn(&common.Project{}, "Command") {
		if err := migrator.DropColumn(&common.Project{}, "Command"); err != nil {
			return err
		}
	}
	return nil
}
