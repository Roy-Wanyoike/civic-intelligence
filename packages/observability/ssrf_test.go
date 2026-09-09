package observability

import "testing"

func TestIsURLAllowed_AllowedHost(t *testing.T) {
	tests := []string{
		"https://new.kenyalaw.org/bills/",
		"https://www.parliament.go.ke/the-national-assembly/bills",
		"https://www.president.go.ke/speeches",
		"https://treasury.go.ke",
	}
	for _, url := range tests {
		if err := IsURLAllowed(url); err != nil {
			t.Errorf("expected %s to be allowed: %v", url, err)
		}
	}
}

func TestIsURLAllowed_BlockedHost(t *testing.T) {
	tests := []string{
		"https://evil.com",
		"https://localhost",
		"https://127.0.0.1",
		"https://10.0.0.1",
		"https://192.168.1.1",
		"ftp://kenyalaw.org", // wrong scheme
	}
	for _, url := range tests {
		if err := IsURLAllowed(url); err == nil {
			t.Errorf("expected %s to be blocked, but it was allowed", url)
		}
	}
}

func TestIsURLAllowed_InvalidURL(t *testing.T) {
	if err := IsURLAllowed("not a url"); err == nil {
		t.Error("expected error for invalid URL")
	}
}
