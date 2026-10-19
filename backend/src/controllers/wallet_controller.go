package controllers

import (
	"context"

	"techguild-backend/src/config"
	"techguild-backend/src/dto"
	"techguild-backend/src/services"
	"techguild-backend/src/utils"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type WalletController struct {
	svc *services.WalletService
}

func NewWalletController(cfg *config.Config) *WalletController {
	return &WalletController{
		svc: services.NewWalletService(cfg),
	}
}

// ============================================================
// Get Wallet
// ============================================================

func (c *WalletController) GetWalletHandler(
	ctx context.Context,
	input *dto.GetWalletInput,
) (*dto.GetWalletOutput, error) {
	userIDStr, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid user id")
	}

	wallet, err := c.svc.GetWallet(userID)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	// Recent transactions (5)
	txs, _, _ := c.svc.ListTransactions(userID, 5, 0)
	items := make([]dto.WalletTransactionItem, 0, len(txs))
	for _, t := range txs {
		items = append(items, dto.WalletTransactionItem{
			ID:           t.ID.String(),
			Type:         string(t.Type),
			Amount:       utils.PaiseToRupees(t.AmountMinor),
			AmountMinor:  t.AmountMinor,
			Currency:     t.Currency,
			BalanceAfter: utils.PaiseToRupees(t.BalanceAfterMinor),
			Reference:    t.Reference,
			Description:  t.Description,
			CreatedAt:    t.CreatedAt,
		})
	}

	return &dto.GetWalletOutput{
		Body: dto.WalletResponse{
			WalletID:              wallet.ID.String(),
			AvailableBalance:      utils.PaiseToRupees(wallet.AvailableBalanceMinor),
			PendingBalance:        utils.PaiseToRupees(wallet.PendingBalanceMinor),
			TotalEarned:           utils.PaiseToRupees(wallet.TotalEarnedMinor),
			TotalWithdrawn:        utils.PaiseToRupees(wallet.TotalWithdrawnMinor),
			AvailableBalanceMinor: wallet.AvailableBalanceMinor,
			PendingBalanceMinor:   wallet.PendingBalanceMinor,
			TotalEarnedMinor:      wallet.TotalEarnedMinor,
			TotalWithdrawnMinor:   wallet.TotalWithdrawnMinor,
			Currency:              wallet.Currency,
			RecentTransactions:    items,
		},
	}, nil
}

// ============================================================
// List Wallet Transactions
// ============================================================

func (c *WalletController) ListTransactionsHandler(
	ctx context.Context,
	input *dto.ListWalletTransactionsInput,
) (*dto.ListWalletTransactionsOutput, error) {
	userIDStr, err := utils.GetUserIDFromHumaContext(ctx)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error())
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid user id")
	}

	txs, total, err := c.svc.ListTransactions(userID, input.Limit, input.Offset)
	if err != nil {
		return nil, huma.Error500InternalServerError(err.Error())
	}

	items := make([]dto.WalletTransactionItem, 0, len(txs))
	for _, t := range txs {
		items = append(items, dto.WalletTransactionItem{
			ID:           t.ID.String(),
			Type:         string(t.Type),
			Amount:       utils.PaiseToRupees(t.AmountMinor),
			AmountMinor:  t.AmountMinor,
			Currency:     t.Currency,
			BalanceAfter: utils.PaiseToRupees(t.BalanceAfterMinor),
			Reference:    t.Reference,
			Description:  t.Description,
			CreatedAt:    t.CreatedAt,
		})
	}

	return &dto.ListWalletTransactionsOutput{
		Body: dto.ListWalletTransactionsResponse{
			Transactions: items,
			Total:        total,
			Limit:        input.Limit,
			Offset:       input.Offset,
		},
	}, nil
}
