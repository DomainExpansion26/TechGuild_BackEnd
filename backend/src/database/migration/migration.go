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

		// Projects (DISABLED — replaced by Quest flow)
		// &models.Project{},
		// &models.ProjectSkill{},
		// &models.ProjectAttachment{},
		// &models.ProjectApplication{},
		// &models.ProjectContract{},
		// &models.ProjectMilestone{},
		// &models.ProjectSubmission{},

		// Quests
		&models.Quest{},
		&models.QuestMilestone{},
		&models.QuestSkill{},
		&models.QuestChatChannel{},
		&models.MilestoneRevision{},

		// Party Collaboration
		&models.Party{},
		&models.PartyMember{},
		&models.PartyInvitation{},
		&models.PartyPortfolio{},
		&models.PartySkill{},

		// Task Management (Client TMD)
		&models.Task{},
		&models.QuestActivity{},
		&models.QuestReview{},
		&models.MilestoneApprovalLog{},

		// 2FA Authentication
		&models.UserTwoFactorAuthentication{},
		&models.UserRecoveryCode{},

		// Payment
		&models.PaymentCustomer{},
		&models.ClientPaymentMethod{},
		&models.PayoutMethod{},
		&models.PaymentWebhookEvent{},

		// Quest-based payments
		&models.Payment{},
		&models.MilestonePayment{},
		&models.EscrowTransaction{},

		&models.PaymentTransaction{},
		&models.PlatformFee{},
		&models.Payout{},
		&models.Refund{},

		// Wallet
		&models.Wallet{},
		&models.WalletTransaction{},
	)

	if err != nil {
		log.Fatal("Migration Failed:", err)
	}

	// Raw SQL: CHECK constraints + partial unique indexes
	if err := migratePaymentConstraints(postgres.DB); err != nil {
		log.Fatal("Payment constraints migration failed:", err)
	}

	if err := migrateMilestonePaymentConstraints(postgres.DB); err != nil {
		log.Fatal("Milestone payment constraints migration failed:", err)
	}

	log.Println("Database Migrated Successfully")
}

// ============================================================
// Payment methods + webhook constraints (unchanged)
// ============================================================

func migratePaymentConstraints(db *gorm.DB) error {
	statements := []string{

		// ---------- client_payment_methods ----------
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_pm_type') THEN
				ALTER TABLE client_payment_methods
				ADD CONSTRAINT chk_client_pm_type
				CHECK (type IN ('card','upi','netbanking'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_client_pm_status') THEN
				ALTER TABLE client_payment_methods
				ADD CONSTRAINT chk_client_pm_status
				CHECK (status IN ('active','expired','disabled','failed'));
			END IF;
		END $$;`,

		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_primary_client_pm
    		ON client_payment_methods (user_id, type)
    		WHERE is_primary = TRUE AND deleted_at IS NULL;`,

		// ---------- payout_methods ----------
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_type') THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_type
				CHECK (type IN ('bank','upi','card','wallet'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_status') THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_status
				CHECK (status IN ('pending_verification','verified','failed','disabled'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_payout_method_purpose') THEN
				ALTER TABLE payout_methods
				ADD CONSTRAINT chk_payout_method_purpose
				CHECK (purpose IN ('payout'));
			END IF;
		END $$;`,

		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_primary_payout_method
    		ON payout_methods (user_id, purpose, type)
    		WHERE is_primary = TRUE AND deleted_at IS NULL;`,

		// ---------- payment_webhook_events ----------
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_webhook_event_status') THEN
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

// ============================================================
// Quest-based payment flow constraints (DUAL-FEE MODEL)
// ============================================================

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

		// ==================== milestone_payments (DUAL FEE) ====================

		// Drop old constraints that reference removed fields (gross_amount, platform_fee, freelancer_amount, fee_rate_bps)
		`ALTER TABLE milestone_payments DROP CONSTRAINT IF EXISTS chk_milestone_amount_split;`,
		`ALTER TABLE milestone_payments DROP CONSTRAINT IF EXISTS chk_milestone_amounts_nonneg;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_payment_status') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_payment_status
				CHECK (status IN ('pending','funded','release_pending','released','refund_pending','refunded','disputed','failed'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_fee_type') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_fee_type
				CHECK (fee_type IN ('percentage','fixed'));
			END IF;
		END $$;`,

		// CRITICAL: Dual-fee invariants
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_client_charged') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_client_charged
				CHECK (client_charged_minor = milestone_amount_minor + client_fee_minor);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_freelancer_net') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_freelancer_net
				CHECK (freelancer_net_minor = milestone_amount_minor - freelancer_fee_minor);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_platform_total') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_platform_total
				CHECK (platform_total_minor = client_fee_minor + freelancer_fee_minor);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_balance') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_balance
				CHECK (client_charged_minor = freelancer_net_minor + platform_total_minor);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_milestone_amounts_nonneg') THEN
				ALTER TABLE milestone_payments ADD CONSTRAINT chk_milestone_amounts_nonneg
				CHECK (
					milestone_amount_minor >= 0 AND
					client_fee_minor >= 0 AND
					freelancer_fee_minor >= 0 AND
					client_charged_minor >= 0 AND
					freelancer_net_minor >= 0 AND
					platform_total_minor >= 0 AND
					client_fee_rate_bps >= 0 AND
					freelancer_fee_rate_bps >= 0
				);
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
			IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tx_type') THEN
				ALTER TABLE payment_transactions DROP CONSTRAINT chk_tx_type;
			END IF;
			ALTER TABLE payment_transactions ADD CONSTRAINT chk_tx_type
			CHECK (transaction_type IN (
				'payment','capture','escrow_hold','escrow_release','escrow_refund',
				'platform_fee_client','platform_fee_freelancer',
				'payout','wallet_credit','withdrawal',
				'refund','chargeback','adjustment','fx_conversion'
			));
		END $$;`,

		// ==================== platform_fees ====================

		// Drop old unique index on milestone_payment_id (now 2 rows per mp)
		`DROP INDEX IF EXISTS idx_platform_fees_milestone_payment_id;`,

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
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_platform_fee_side') THEN
				ALTER TABLE platform_fees ADD CONSTRAINT chk_platform_fee_side
				CHECK (side IN ('client','freelancer'));
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_platform_fee_amount_nonneg') THEN
				ALTER TABLE platform_fees ADD CONSTRAINT chk_platform_fee_amount_nonneg
				CHECK (amount >= 0 AND rate_bps >= 0);
			END IF;
		END $$;`,

		// Unique constraint: one row per (milestone_payment, side)
		`CREATE UNIQUE INDEX IF NOT EXISTS uniq_platform_fee_per_side
			ON platform_fees (milestone_payment_id, side);`,

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

		// ==================== indexes ====================

		`CREATE INDEX IF NOT EXISTS idx_payments_user_status
			ON payments (user_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_escrow_quest_milestone_status
			ON escrow_transactions (quest_milestone_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_payouts_user_status
			ON payouts (user_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_refunds_payment_status
			ON refunds (payment_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_ledger_payment_created
			ON payment_transactions (payment_id, created_at DESC);`,

		`CREATE INDEX IF NOT EXISTS idx_milestone_payments_quest
			ON milestone_payments (quest_id, status);`,

		`CREATE INDEX IF NOT EXISTS idx_payment_transactions_quest_milestone
			ON payment_transactions (quest_milestone_id, created_at DESC);`,

		// ==================== wallets (Phase 5) ====================

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_wallet_balances_nonneg') THEN
				ALTER TABLE wallets ADD CONSTRAINT chk_wallet_balances_nonneg
				CHECK (
					available_balance_minor >= 0 AND
					pending_balance_minor >= 0 AND
					total_earned_minor >= 0 AND
					total_withdrawn_minor >= 0
				);
			END IF;
		END $$;`,

		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_wallet_tx_type') THEN
				ALTER TABLE wallet_transactions ADD CONSTRAINT chk_wallet_tx_type
				CHECK (type IN ('credit','debit','hold','hold_release','withdrawal_failed'));
			END IF;
		END $$;`,

		`CREATE INDEX IF NOT EXISTS idx_wallet_tx_user_created
			ON wallet_transactions (user_id, created_at DESC);`,

		`CREATE INDEX IF NOT EXISTS idx_wallet_tx_reference
			ON wallet_transactions (reference);`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
