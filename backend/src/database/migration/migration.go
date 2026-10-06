package migration

import (
	"log"

	"techguild-backend/src/database/postgres"
	"techguild-backend/src/models"

	"gorm.io/gorm"
)

func Migrate() {

	err := postgres.DB.AutoMigrate(

		// Users & Profiles
		&models.User{},
		&models.IndividualProfile{},
		&models.AgencyProfile{},
		&models.ClientProfile{},
		&models.UserSession{},

		// Verification
		&models.VerificationRecord{},
		&models.VerificationDocument{},
		&models.GovernmentID{},
		&models.GovtIDDedup{},
		&models.BusinessPANDedup{},

		// Rules & Notifications
		&models.RuleDocument{},
		&models.PolicyChangeNotification{},

		// Audit
		&models.AuditLog{},

		// Projects
		&models.Project{},
		&models.ProjectSkill{},
		&models.ProjectAttachment{},
		&models.ProjectApplication{},
		&models.ProjectContract{},
		&models.ProjectMilestone{},
		&models.ProjectSubmission{},

		// Quests (Gamified Client Creation)
		&models.Quest{},
		&models.QuestMilestone{},
		&models.QuestSkill{},

		// Team Collaboration
		&models.Team{},
		&models.TeamMember{},
		&models.TeamInvitation{},
		&models.TeamPortfolio{},
		&models.TeamSkill{},

		// 2FA Authentication
		&models.UserTwoFactorAuthentication{},
		&models.UserRecoveryCode{},

		// Payment
		&models.PaymentCustomer{},
		&models.ClientPaymentMethod{},
		&models.PayoutMethod{},
		&models.PaymentWebhookEvent{},

		&models.Payment{},
		&models.MilestonePayment{},
		&models.EscrowTransaction{},

		&models.PaymentTransaction{},
		&models.PlatformFee{},
		&models.Payout{},
		&models.Refund{},
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	// Raw SQL: CHECK constraints + partial unique indexes
	// (AutoMigrate GORM tags se ye nahi banti)
	if err := migratePaymentConstraints(postgres.DB); err != nil {
		log.Fatal("Payment constraints migration failed:", err)
	}

	// Milestone flow constraints + indexes
	if err := migrateMilestonePaymentConstraints(postgres.DB); err != nil {
		log.Fatal("Milestone payment constraints migration failed:", err)
	}

	log.Println("Database Migrated Successfully")
}

// migratePaymentConstraints adds CHECK constraints and partial unique indexes
// that GORM AutoMigrate cannot express via struct tags.
//
// Ye function idempotent hai — dobara chalane par error nahi dega.
func migratePaymentConstraints(db *gorm.DB) error {

	statements := []string{

		// ---------- client_payment_methods ----------

		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_pm_type'
			) THEN
				ALTER TABLE client_payment_methods
				ADD CONSTRAINT chk_client_pm_type
				CHECK (type IN ('card','upi','netbanking'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_pm_status'
			) THEN
				ALTER TABLE client_payment_methods
				ADD CONSTRAINT chk_client_pm_status
				CHECK (status IN ('active','expired','disabled','failed'));
			END IF;
		END $$;`,

		// One primary payment method per (user, type)
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_primary_client_pm
    		ON client_payment_methods (user_id, type)
    		WHERE is_primary = TRUE AND deleted_at IS NULL;`,

		// ---------- payout_methods ----------

		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_type'
			) THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_type
				CHECK (type IN ('bank','upi','card','wallet'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_status'
			) THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_status
				CHECK (status IN ('pending_verification','verified','failed','disabled'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_purpose'
			) THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_purpose
				CHECK (purpose IN ('payout'));
			END IF;
		END $$;`,

		// One primary payout method per (user, purpose, type)
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_primary_payout_method
    		ON payout_methods (user_id, purpose, type)
    		WHERE is_primary = TRUE AND deleted_at IS NULL;`,

		// ---------- payment_webhook_events ----------
		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_webhook_event_status'
			) THEN
				ALTER TABLE payment_webhook_events
				ADD CONSTRAINT chk_webhook_event_status
				CHECK (status IN ('received','processing','processed','failed'));
			END IF;
		END $$;`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}

// migrateMilestonePaymentConstraints adds CHECK constraints and extra indexes
// for the 7 payment-flow tables. Ye v3 spec ke financial invariants enforce
// karta hai — bina inke data corruption possible hai.
//
// Idempotent hai — dobara chalane pe error nahi dega.
func migrateMilestonePaymentConstraints(db *gorm.DB) error {
	statements := []string{

		// ==================== payments ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payment_status') THEN
				ALTER TABLE payments ADD CONSTRAINT chk_payment_status
				CHECK (status IN ('pending','authorized','captured','failed','cancelled','refunded','partially_refunded'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payment_amount_positive') THEN
				ALTER TABLE payments ADD CONSTRAINT chk_payment_amount_positive
				CHECK (amount > 0);
			END IF;
		END $$;`,

		// ==================== milestone_payments ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_payment_status') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_payment_status
				CHECK (status IN ('pending','funded','awaiting_approval','approved','release_pending','released','refund_pending','refunded','disputed','failed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_fee_type') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_fee_type
				CHECK (fee_type IN ('percentage','fixed'));
			END IF;
		END $$;`,

		// CRITICAL: gross = platform_fee + freelancer_amount (v3 spec invariant)
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_amount_split') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_amount_split
				CHECK (gross_amount = platform_fee + freelancer_amount);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_amounts_nonneg') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_amounts_nonneg
				CHECK (gross_amount >= 0 AND platform_fee >= 0 AND freelancer_amount >= 0 AND fee_rate_bps >= 0);
			END IF;
		END $$;`,

		// ==================== escrow_transactions ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_escrow_status') THEN
				ALTER TABLE escrow_transactions ADD CONSTRAINT chk_escrow_status
				CHECK (status IN ('pending','held','release_pending','released','refund_pending','refunded','failed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_escrow_amount_positive') THEN
				ALTER TABLE escrow_transactions ADD CONSTRAINT chk_escrow_amount_positive
				CHECK (amount > 0);
			END IF;
		END $$;`,

		// ==================== payment_transactions (ledger) ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tx_type') THEN
				ALTER TABLE payment_transactions ADD CONSTRAINT chk_tx_type
				CHECK (transaction_type IN ('payment','capture','escrow_hold','escrow_release','platform_fee','payout','refund','chargeback','adjustment','fx_conversion'));
			END IF;
		END $$;`,

		// ==================== platform_fees ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_platform_fee_status') THEN
				ALTER TABLE platform_fees ADD CONSTRAINT chk_platform_fee_status
				CHECK (status IN ('pending','accrued','settled','reversed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_platform_fee_type') THEN
				ALTER TABLE platform_fees ADD CONSTRAINT chk_platform_fee_type
				CHECK (fee_type IN ('percentage','fixed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_platform_fee_amount_nonneg') THEN
				ALTER TABLE platform_fees ADD CONSTRAINT chk_platform_fee_amount_nonneg
				CHECK (amount >= 0 AND rate_bps >= 0);
			END IF;
		END $$;`,

		// ==================== payouts ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_status') THEN
				ALTER TABLE payouts ADD CONSTRAINT chk_payout_status
				CHECK (status IN ('pending','processing','paid','failed','cancelled','reversed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_amount_positive') THEN
				ALTER TABLE payouts ADD CONSTRAINT chk_payout_amount_positive
				CHECK (amount > 0);
			END IF;
		END $$;`,

		// ==================== refunds ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_refund_status') THEN
				ALTER TABLE refunds ADD CONSTRAINT chk_refund_status
				CHECK (status IN ('pending','processing','completed','failed','cancelled'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_refund_amount_positive') THEN
				ALTER TABLE refunds ADD CONSTRAINT chk_refund_amount_positive
				CHECK (amount > 0);
			END IF;
		END $$;`,

		// ==================== extra composite indexes ====================

		`CREATE INDEX IF NOT EXISTS idx_payments_user_status
			ON payments (user_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_escrow_milestone_status
			ON escrow_transactions (milestone_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_payouts_user_status
			ON payouts (user_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_refunds_payment_status
			ON refunds (payment_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_ledger_payment_created
			ON payment_transactions (payment_id, created_at DESC);`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}
