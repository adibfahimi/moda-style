package common

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// testJWTSecret is the signing key every test in this package uses. It is set
// in the environment by TestMain before any test runs.
const testJWTSecret = "common-package-test-secret"

// TestMain installs the environment the package needs. The JWT signing key is
// resolved lazily on first use, so setting JWT_SECRET here guarantees each test
// signs and verifies tokens with a known value.
func TestMain(m *testing.M) {
	os.Setenv("JWT_SECRET", testJWTSecret)
	os.Exit(m.Run())
}

// performRequest drives a Fiber app with the given HTTP method, path, optional
// raw body and headers, returning the response and its raw bytes.
//
// It fails the test immediately if the request cannot be served.
func performRequest(t *testing.T, app *fiber.App, method, path, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test(%s %s) failed: %v", method, path, err)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body failed: %v", err)
	}

	return resp, raw
}

// decodeJSON unmarshals raw JSON bytes into a generic map so tests can assert on
// individual fields without declaring a struct per response shape.
func decodeJSON(t *testing.T, raw []byte) map[string]interface{} {
	t.Helper()

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("response body is not valid JSON (%q): %v", string(raw), err)
	}
	return decoded
}

// errorMessage extracts the "error" field from a JSON error envelope produced by
// SendErrorResponse.
func errorMessage(t *testing.T, raw []byte) string {
	t.Helper()

	message, ok := decodeJSON(t, raw)["error"].(string)
	if !ok {
		t.Fatalf("response does not contain a string \"error\" field: %s", string(raw))
	}
	return message
}
