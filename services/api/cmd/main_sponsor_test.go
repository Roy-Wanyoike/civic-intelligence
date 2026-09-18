package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleMpesaSponsor_Returns501 verifies the M-Pesa sponsor endpoint
// fails closed with 501 Not Implemented until the Daraja integration is
// wired. Closes GAP-53-1 — the endpoint previously returned a fake
// "pending" success response without contacting Safaricom.
func TestHandleMpesaSponsor_Returns501(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sponsor/mpesa", strings.NewReader(`{"phone":"0712345678","amount":500}`))
	rr := httptest.NewRecorder()
	handleMpesaSponsor(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rr.Code)
	}

	var resp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "not_implemented" {
		t.Errorf("expected error=not_implemented, got %q", resp.Error)
	}
	if !strings.Contains(resp.Message, "do not use in production") {
		t.Errorf("expected message to warn against production use, got %q", resp.Message)
	}
}

// TestHandleCardSponsor_Returns501 verifies the Stripe Checkout sponsor
// endpoint fails closed with 501 Not Implemented until Stripe is wired.
// Closes GAP-53-1.
func TestHandleCardSponsor_Returns501(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sponsor/card", strings.NewReader(`{"amount":500,"email":"you@example.com"}`))
	rr := httptest.NewRecorder()
	handleCardSponsor(rr, req)

	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "do not use in production") {
		t.Errorf("expected warning in body, got %s", rr.Body.String())
	}
}

// TestHandleMpesaSponsor_MethodNotAllowed verifies the 501 path is only
// hit for POST — wrong methods are rejected earlier with 405.
func TestHandleMpesaSponsor_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sponsor/mpesa", nil)
	rr := httptest.NewRecorder()
	handleMpesaSponsor(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET, got %d", rr.Code)
	}
}

// TestHandleMpesaCallback_Returns501 verifies the M-Pesa callback also
// fails closed — without the Daraja integration, we cannot authenticate
// the callback as a genuine Safaricom request.
func TestHandleMpesaCallback_Returns501(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sponsor/mpesa/callback", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handleMpesaCallback(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
}

// TestHandleStripeWebhook_Returns501 verifies the Stripe webhook also
// fails closed — without STRIPE_SECRET_KEY, the webhook signature cannot
// be verified, so accepting the request would let an attacker forge
// payment confirmations.
func TestHandleStripeWebhook_Returns501(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sponsor/card/webhook", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handleStripeWebhook(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
}
