package reviews

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "Role ID", Kind: "uuid"},
	{Key: "productId", Label: "Product ID", Kind: "uuid"},
	{Key: "rating", Label: "Rating", Kind: "number"},
	{Key: "comment", Label: "Review", Kind: "string", Min: 1, Max: 255, Nullable: true},
}

var updateFields = []promotionutil.Field{
	{Key: "comment", Label: "Review", Kind: "string", Min: 1, Max: 255, Nullable: true},
	{Key: "rating", Label: "Rating", Kind: "number"},
}

func checkRating(rating float64) error {
	if rating >= 1 && rating <= 5 {
		return nil
	}
	tooSmall := rating < 1
	code, bound, message := "too_big", 5, "🚀 Whoa! Rating is too long. Maximum allowed is 5 characters! ✍️ {Maximum Length}"
	if tooSmall {
		code, bound, message = "too_small", 1, "🔡 Rating must be at least 1 characters. Keep going! 💪 {Minimum Length}"
	}
	issue := map[string]any{"code": code, "origin": "number", "inclusive": true, "message": message, "path": []string{"rating"}}
	if tooSmall {
		issue["minimum"] = bound
	} else {
		issue["maximum"] = bound
	}
	return httpx.ZodIssue(issue)
}
