package utils

import (
	"errors"
	"fmt"

	"techguild-backend/src/models"
)

const MinQuestBudgetINR = 1000.0

var ErrBudgetBelowMinimum = errors.New("quest budget must be at least ₹1,000")

type RankTier struct {
	Rank               string  `json:"rank"`
	PreviousCumulative float64 `json:"previous_cumulative"`
	AdditionalAmount   float64 `json:"additional_amount"`
	TotalThreshold     float64 `json:"total_threshold"`
	Description        string  `json:"description"`
}

var RankTiers = []RankTier{
	{Rank: models.RankF, PreviousCumulative: 0, AdditionalAmount: 10000, TotalThreshold: 10000, Description: "₹1,000 – ₹10,000"},
	{Rank: models.RankE, PreviousCumulative: 10000, AdditionalAmount: 15000, TotalThreshold: 25000, Description: "Up to ₹25,000"},
	{Rank: models.RankD, PreviousCumulative: 25000, AdditionalAmount: 20000, TotalThreshold: 45000, Description: "Up to ₹45,000"},
	{Rank: models.RankC, PreviousCumulative: 45000, AdditionalAmount: 25000, TotalThreshold: 70000, Description: "Up to ₹70,000"},
	{Rank: models.RankB, PreviousCumulative: 70000, AdditionalAmount: 30000, TotalThreshold: 100000, Description: "Up to ₹1,00,000"},
	{Rank: models.RankA, PreviousCumulative: 100000, AdditionalAmount: 35000, TotalThreshold: 135000, Description: "Up to ₹1,35,000"},
	{Rank: models.RankS, PreviousCumulative: 135000, AdditionalAmount: 40000, TotalThreshold: 175000, Description: "Up to ₹1,75,000"},
	{Rank: models.RankSS, PreviousCumulative: 175000, AdditionalAmount: 45000, TotalThreshold: 220000, Description: "Up to ₹2,20,000"},
	{Rank: models.RankSSS, PreviousCumulative: 220000, AdditionalAmount: 50000, TotalThreshold: 270000, Description: "Above ₹2,20,000"},
}

// GetMaxBudgetForRank returns the maximum allowed quest budget for a given client rank.
func GetMaxBudgetForRank(rank string) (float64, error) {
	for _, tier := range RankTiers {
		if tier.Rank == rank {
			return tier.TotalThreshold, nil
		}
	}
	return 0, fmt.Errorf("invalid rank: %s", rank)
}

// ValidateBudgetForRank checks if the budget satisfies:
// 1. Minimum budget cap of ₹1,000 across all ranks.
// 2. Maximum budget cap allowed for the client's respective rank.
func ValidateBudgetForRank(clientRank string, budgetINR float64) error {
	if budgetINR < MinQuestBudgetINR {
		return ErrBudgetBelowMinimum
	}

	if clientRank == "" {
		clientRank = models.RankF
	}

	// SSS has no ceiling cap (above ₹2,20,000)
	if clientRank == models.RankSSS {
		return nil
	}

	maxBudget, err := GetMaxBudgetForRank(clientRank)
	if err != nil {
		return err
	}

	if budgetINR > maxBudget {
		return fmt.Errorf("budget ₹%.2f exceeds the maximum allowed limit of ₹%.2f for client rank %s", budgetINR, maxBudget, clientRank)
	}

	return nil
}

// ComputeRequiredMinRank calculates the required minimum rank based on the cumulative 9-tier progression matrix.
// Quests must have a minimum budget of ₹1,000.
func ComputeRequiredMinRank(budgetINR float64) (string, error) {
	if budgetINR < MinQuestBudgetINR {
		return "", ErrBudgetBelowMinimum
	}

	switch {
	case budgetINR <= 10000:
		return models.RankF, nil
	case budgetINR <= 25000:
		return models.RankE, nil
	case budgetINR <= 45000:
		return models.RankD, nil
	case budgetINR <= 70000:
		return models.RankC, nil
	case budgetINR <= 100000:
		return models.RankB, nil
	case budgetINR <= 135000:
		return models.RankA, nil
	case budgetINR <= 175000:
		return models.RankS, nil
	case budgetINR <= 220000:
		return models.RankSS, nil
	default:
		return models.RankSSS, nil
	}
}
