package recommendation

import (
	"context"

	"github.com/jourloy/somivyn/internal/fit"
	"github.com/jourloy/somivyn/internal/product"
)

type Recommendation struct {
	Type    string  `json:"type"`    // "success", "warning", "info", "tip"
	Message string  `json:"message"` // Translation key
	Icon    string  `json:"icon"`    // Icon name for frontend
	Value   float64 `json:"value"`   // Numeric value for interpolation
}

type Service interface {
	GetDailyRecommendations(ctx context.Context, uid string, today string) ([]Recommendation, error)
}

type service struct {
	productService product.Service
	fitService     fit.Service
}

func NewService() Service {
	return &service{
		productService: product.NewService(),
		fitService:     fit.NewService(),
	}
}

func (s *service) GetDailyRecommendations(ctx context.Context, uid string, today string) ([]Recommendation, error) {
	// Get user's fit profile
	fitProfile, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	// Get today's products
	products, err := s.productService.GetAllByToday(ctx, uid, today)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	var totalCalories, totalProtein, totalFat, totalCarbs float64
	for _, p := range products {
		if !p.IsWater {
			totalCalories += p.Calories
			totalProtein += p.Protein
			totalFat += p.Fat
			totalCarbs += p.Carbs
		}
	}

	// Calculate percentages
	caloriePercent := (totalCalories / fitProfile.Calories) * 100
	proteinPercent := (totalProtein / fitProfile.Protein) * 100

	var recommendations []Recommendation

	// Calorie recommendations
	if caloriePercent < 50 {
		remaining := fitProfile.Calories - totalCalories
		recommendations = append(recommendations, Recommendation{
			Type:    "info",
			Message: "recommendations.calories.remaining",
			Icon:    "flame",
			Value:   remaining,
		})
	} else if caloriePercent >= 90 && caloriePercent <= 110 {
		recommendations = append(recommendations, Recommendation{
			Type:    "success",
			Message: "recommendations.calories.perfect",
			Icon:    "check-circle",
			Value:   caloriePercent,
		})
	} else if caloriePercent > 110 {
		recommendations = append(recommendations, Recommendation{
			Type:    "warning",
			Message: "recommendations.calories.exceeded",
			Icon:    "alert-triangle",
			Value:   caloriePercent,
		})
	}

	// Protein recommendations
	if proteinPercent < 70 {
		recommendations = append(recommendations, Recommendation{
			Type:    "tip",
			Message: "recommendations.protein.increase",
			Icon:    "diamond",
			Value:   proteinPercent,
		})
	} else if proteinPercent >= 80 && proteinPercent <= 120 {
		recommendations = append(recommendations, Recommendation{
			Type:    "success",
			Message: "recommendations.protein.great",
			Icon:    "check-circle",
			Value:   proteinPercent,
		})
	}

	// Water recommendations
	if fitProfile.WaterLimit != nil {
		var totalWater float64
		for _, p := range products {
			if p.IsWater {
				totalWater += float64(p.Amount)
			}
		}
		waterPercent := (totalWater / float64(*fitProfile.WaterLimit)) * 100
		if waterPercent < 50 {
			recommendations = append(recommendations, Recommendation{
				Type:    "info",
				Message: "recommendations.water.increase",
				Icon:    "droplet",
				Value:   waterPercent,
			})
		}
	}

	// Meal balance recommendations
	mealCounts := make(map[string]int)
	for _, p := range products {
		if !p.IsWater && p.MealType != nil {
			mealCounts[*p.MealType]++
		}
	}
	if mealCounts["breakfast"] == 0 && len(products) > 0 {
		recommendations = append(recommendations, Recommendation{
			Type:    "tip",
			Message: "recommendations.meals.noBreakfast",
			Icon:    "coffee",
			Value:   0,
		})
	}

	// Limit to top 3 recommendations
	if len(recommendations) > 3 {
		recommendations = recommendations[:3]
	}

	return recommendations, nil
}
