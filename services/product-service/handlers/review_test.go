package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/adibfahimi/moda-style/services/product-service/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// seedReview inserts a review directly so listing tests do not depend on the
// create endpoint.
func seedReview(t *testing.T, db *gorm.DB, productID, userID uint, userName string, rating int) models.Review {
	t.Helper()

	review := models.Review{
		ProductID: productID,
		UserID:    userID,
		UserName:  userName,
		Rating:    rating,
		Comment:   "Great quality and true to size.",
	}
	if err := db.Create(&review).Error; err != nil {
		t.Fatalf("creating review: %v", err)
	}
	return review
}

func TestGetProductReviewsReportsSummaryAndPagination(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	seedReview(t, db, seed.dress.ID, 1, "Ada", 5)
	seedReview(t, db, seed.dress.ID, 2, "Grace", 4)

	status, raw := request(t, app, http.MethodGet, fmt.Sprintf("/api/v1/products/%d/reviews", seed.dress.ID), "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if total := numberField(t, body, "total"); total != 2 {
		t.Errorf("expected total 2, got %v", total)
	}
	if avg := numberField(t, body, "average_rating"); avg != 4.5 {
		t.Errorf("expected average rating 4.5, got %v", avg)
	}

	reviews := sliceField(t, body, "reviews")
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d: %s", len(reviews), raw)
	}
	first := reviews[0].(map[string]interface{})
	if name := stringField(t, first, "user_name"); name == "" {
		t.Error("expected the review author name to be returned")
	}
}

func TestGetProductReviewsPaginates(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	seedReview(t, db, seed.dress.ID, 1, "Ada", 5)
	seedReview(t, db, seed.dress.ID, 2, "Grace", 4)
	seedReview(t, db, seed.dress.ID, 3, "Alan", 3)

	status, raw := request(t, app, http.MethodGet,
		fmt.Sprintf("/api/v1/products/%d/reviews?page=2&limit=2", seed.dress.ID), "", "")
	if status != fiber.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if reviews := sliceField(t, body, "reviews"); len(reviews) != 1 {
		t.Errorf("expected 1 review on the second page, got %d: %s", len(reviews), raw)
	}
	if total := numberField(t, body, "total"); total != 3 {
		t.Errorf("total must ignore pagination, got %v", total)
	}
}

func TestGetProductReviewsReturnsNotFoundForUnknownProduct(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodGet, "/api/v1/products/4242/reviews", "", "")
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Product not found" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestCreateReviewRequiresAuthentication(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodPost, "/api/v1/products/1/reviews",
		`{"rating":5,"comment":"Great quality and true to size."}`, "")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Authorization header required" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestCreateReviewStoresAuthorNameFromToken(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/products/%d/reviews", seed.dress.ID),
		`{"rating":5,"comment":"Great quality and true to size."}`, tokenFor(t, 7, "Ada Lovelace"))
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	body := decodeJSON(t, raw)
	if msg := stringField(t, body, "message"); msg != "Review submitted successfully" {
		t.Errorf("unexpected message %q", msg)
	}
	if rating := numberField(t, objectField(t, body, "review"), "rating"); rating != 5 {
		t.Errorf("expected rating 5, got %v", rating)
	}

	var stored models.Review
	if err := db.Where("product_id = ? AND user_id = ?", seed.dress.ID, 7).First(&stored).Error; err != nil {
		t.Fatalf("loading stored review: %v", err)
	}
	if stored.UserName != "Ada Lovelace" {
		t.Errorf("expected the token user name to be stored, got %q", stored.UserName)
	}
}

func TestCreateReviewFallsBackToAnonymousAuthorName(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/products/%d/reviews", seed.dress.ID),
		`{"rating":4,"comment":"Nice fabric, slightly large."}`, tokenFor(t, 8, ""))
	if status != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", status, raw)
	}

	var stored models.Review
	if err := db.Where("product_id = ? AND user_id = ?", seed.dress.ID, 8).First(&stored).Error; err != nil {
		t.Fatalf("loading stored review: %v", err)
	}
	if stored.UserName != "Anonymous" {
		t.Errorf("expected the anonymous fallback, got %q", stored.UserName)
	}
}

func TestCreateReviewRejectsSecondReviewBySameUser(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)

	seedReview(t, db, seed.dress.ID, 9, "Ada", 5)

	status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/products/%d/reviews", seed.dress.ID),
		`{"rating":4,"comment":"Changed my mind about it."}`, tokenFor(t, 9, "Ada"))
	if status != fiber.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "You have already reviewed this product" {
		t.Errorf("unexpected error message %q", msg)
	}
}

func TestCreateReviewValidatesBody(t *testing.T) {
	db := newTestDB(t)
	app := newTestApp()
	seed := seedCatalog(t, db)
	token := tokenFor(t, 11, "Ada")

	tests := []struct {
		name            string
		body            string
		wantMessagePart string
	}{
		{"rating above five", `{"rating":6,"comment":"Great quality and true to size."}`, "rating"},
		{"missing rating", `{"comment":"Great quality and true to size."}`, "rating"},
		{"short comment", `{"rating":5,"comment":"short"}`, "comment must be at least 10 characters"},
		{"missing comment", `{"rating":5}`, "comment is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := request(t, app, http.MethodPost, fmt.Sprintf("/api/v1/products/%d/reviews", seed.dress.ID), tc.body, token)
			if status != fiber.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", status, raw)
			}
			msg := stringField(t, decodeJSON(t, raw), "error")
			if !strings.Contains(msg, tc.wantMessagePart) {
				t.Errorf("expected error to mention %q, got %q", tc.wantMessagePart, msg)
			}
		})
	}
}

func TestCreateReviewReturnsNotFoundForUnknownProduct(t *testing.T) {
	newTestDB(t)
	app := newTestApp()

	status, raw := request(t, app, http.MethodPost, "/api/v1/products/4242/reviews",
		`{"rating":5,"comment":"Great quality and true to size."}`, tokenFor(t, 12, "Ada"))
	if status != fiber.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", status, raw)
	}
	if msg := stringField(t, decodeJSON(t, raw), "error"); msg != "Product not found" {
		t.Errorf("unexpected error message %q", msg)
	}
}
