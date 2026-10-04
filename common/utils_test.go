package common

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// samplePayload exercises every validation tag that FormatValidationErrors
// knows how to translate, plus one it does not (gte).
type samplePayload struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"min=3"`
	Bio   string `json:"bio" validate:"max=10"`
	Age   int    `json:"age" validate:"gte=18"`
}

// validPayload is a payload that satisfies every tag on samplePayload.
func validPayload() samplePayload {
	return samplePayload{
		Name:  "Ada Lovelace",
		Email: "ada@example.com",
		Code:  "abc",
		Bio:   "short bio",
		Age:   36,
	}
}

func TestValidateStruct_AcceptsValidPayload(t *testing.T) {
	if err := ValidateStruct(validPayload()); err != nil {
		t.Fatalf("ValidateStruct(valid payload) = %v, want nil", err)
	}
}

func TestValidateStruct_RejectsEachInvalidPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     samplePayload
		wantMessage string
	}{
		{
			name:        "missing name",
			payload:     samplePayload{Email: "ada@example.com", Code: "abc", Bio: "short bio", Age: 36},
			wantMessage: "name is required",
		},
		{
			name:        "malformed email",
			payload:     samplePayload{Name: "Ada", Email: "not-an-email", Code: "abc", Bio: "short bio", Age: 36},
			wantMessage: "email must be a valid email",
		},
		{
			name:        "code below the minimum length",
			payload:     samplePayload{Name: "Ada", Email: "ada@example.com", Code: "a", Bio: "short bio", Age: 36},
			wantMessage: "code must be at least 3 characters",
		},
		{
			name:        "bio above the maximum length",
			payload:     samplePayload{Name: "Ada", Email: "ada@example.com", Code: "abc", Bio: "a bio that is far too long", Age: 36},
			wantMessage: "bio must be at most 10 characters",
		},
		{
			name:        "age below the allowed range",
			payload:     samplePayload{Name: "Ada", Email: "ada@example.com", Code: "abc", Bio: "short bio", Age: 17},
			wantMessage: "age is invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(tc.payload)
			if err == nil {
				t.Fatal("ValidateStruct() = nil, want a validation error")
			}
			if got := FormatValidationErrors(err); !strings.Contains(got, tc.wantMessage) {
				t.Errorf("message = %q, want it to contain %q", got, tc.wantMessage)
			}
		})
	}
}

func TestFormatValidationErrors_JoinsMessagesInFieldOrder(t *testing.T) {
	err := ValidateStruct(samplePayload{
		Email: "not-an-email",
		Code:  "a",
		Bio:   "a bio that is far too long",
		Age:   17,
	})
	if err == nil {
		t.Fatal("expected the payload to be invalid")
	}

	want := "name is required, email must be a valid email, " +
		"code must be at least 3 characters, bio must be at most 10 characters, age is invalid"
	if got := FormatValidationErrors(err); got != want {
		t.Errorf("FormatValidationErrors() =\n  %q\nwant\n  %q", got, want)
	}
}

func TestFormatValidationErrors_FallsBackForUnknownErrors(t *testing.T) {
	if got := FormatValidationErrors(errors.New("boom")); got != "validation failed" {
		t.Errorf("FormatValidationErrors(non-validation error) = %q, want %q", got, "validation failed")
	}
}

// parsePayload is the body shape used by the ParseAndValidate tests.
type parsePayload struct {
	Name string `json:"name" validate:"required"`
}

// newParseApp registers a handler that mirrors the "parse then validate"
// pattern every service handler follows.
func newParseApp() *fiber.App {
	app := fiber.New()
	app.Post("/items", func(c *fiber.Ctx) error {
		var body parsePayload
		if !ParseAndValidate(c, &body) {
			return nil
		}
		return SendSuccessResponse(c, fiber.StatusCreated, fiber.Map{"name": body.Name})
	})
	return app
}

func TestParseAndValidate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{name: "valid body", body: `{"name":"Ada"}`, wantStatus: http.StatusCreated},
		{name: "malformed JSON", body: `{`, wantStatus: http.StatusBadRequest, wantError: "Invalid request body"},
		{name: "missing required field", body: `{}`, wantStatus: http.StatusBadRequest, wantError: "name is required"},
	}

	app := newParseApp()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, raw := performRequest(t, app, http.MethodPost, "/items", tc.body, nil)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, tc.wantStatus, raw)
			}

			if tc.wantError == "" {
				if got := decodeJSON(t, raw)["name"]; got != "Ada" {
					t.Errorf("name = %v, want %q", got, "Ada")
				}
				return
			}
			if got := errorMessage(t, raw); got != tc.wantError {
				t.Errorf("error = %q, want %q", got, tc.wantError)
			}
		})
	}
}

func TestSendErrorResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/boom", func(c *fiber.Ctx) error {
		return SendErrorResponse(c, fiber.StatusTeapot, "nope")
	})

	resp, raw := performRequest(t, app, http.MethodGet, "/boom", "", nil)
	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusTeapot)
	}
	if got := errorMessage(t, raw); got != "nope" {
		t.Errorf("error = %q, want %q", got, "nope")
	}
}

func TestSendSuccessResponse(t *testing.T) {
	app := fiber.New()
	app.Post("/echo", func(c *fiber.Ctx) error {
		return SendSuccessResponse(c, http.StatusAccepted, fiber.Map{"ok": true})
	})

	resp, raw := performRequest(t, app, http.MethodPost, "/echo", `{"ignored":true}`, nil)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want it to start with %q", contentType, "application/json")
	}
	if got := decodeJSON(t, raw)["ok"]; got != true {
		t.Errorf("ok = %v, want true", got)
	}
}
