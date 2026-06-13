package signing_test

import (
	"strings"
	"testing"

	"webhook/internal/signing"
)

func TestSign(t *testing.T) {
	secret := "whsec_testsecret"
	deliveryID := "d1e2f3a4-0000-0000-0000-000000000001"
	timestamp := "1700000000"
	body := []byte(`{"order_id":"ord_001","amount":4999}`)

	sig := signing.Sign(secret, deliveryID, timestamp, body)

	if !strings.HasPrefix(sig, "sha256=") {
		t.Fatalf("signature must start with sha256=, got %q", sig)
	}
	if len(sig) != len("sha256=")+64 {
		t.Fatalf("expected sha256=<64 hex chars>, got len %d: %q", len(sig), sig)
	}
}

func TestSign_Deterministic(t *testing.T) {
	secret := "s"
	id := "id"
	ts := "12345"
	body := []byte("body")

	sig1 := signing.Sign(secret, id, ts, body)
	sig2 := signing.Sign(secret, id, ts, body)

	if sig1 != sig2 {
		t.Fatalf("Sign is not deterministic: %q != %q", sig1, sig2)
	}
}

func TestSign_DifferentInputsProduceDifferentSignatures(t *testing.T) {
	secret := "whsec_abc"
	id := "delivery-1"
	ts := "1700000000"
	body := []byte(`{"amount":100}`)

	cases := []struct {
		name    string
		secret  string
		id      string
		ts      string
		body    []byte
	}{
		{"different secret", "whsec_other", id, ts, body},
		{"different id", secret, "delivery-2", ts, body},
		{"different timestamp", secret, id, "9999999999", body},
		{"different body", secret, id, ts, []byte(`{"amount":200}`)},
	}

	baseline := signing.Sign(secret, id, ts, body)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := signing.Sign(tc.secret, tc.id, tc.ts, tc.body)
			if got == baseline {
				t.Errorf("expected different signature but got same as baseline")
			}
		})
	}
}

func TestHashAPIKey(t *testing.T) {
	raw := "whk_" + strings.Repeat("a1b2c3d4", 8) // 64 hex chars — fake test key
	hash := signing.HashAPIKey(raw)

	if len(hash) != 64 {
		t.Fatalf("HashAPIKey must return 64 hex chars, got %d: %q", len(hash), hash)
	}
	for _, c := range hash {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Fatalf("HashAPIKey returned non-hex character %q in %q", c, hash)
		}
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	raw := "whk_abc123"
	h1 := signing.HashAPIKey(raw)
	h2 := signing.HashAPIKey(raw)
	if h1 != h2 {
		t.Fatal("HashAPIKey is not deterministic")
	}
}

func TestHashAPIKey_DoesNotReturnRawKey(t *testing.T) {
	raw := "whk_supersecretkey"
	hash := signing.HashAPIKey(raw)
	if hash == raw {
		t.Fatal("HashAPIKey returned the raw key unchanged")
	}
	if strings.Contains(hash, "supersecret") {
		t.Fatal("raw key content visible in hash output")
	}
}

func TestHashAPIKey_DifferentKeysProduceDifferentHashes(t *testing.T) {
	h1 := signing.HashAPIKey("whk_aaa")
	h2 := signing.HashAPIKey("whk_bbb")
	if h1 == h2 {
		t.Fatal("different keys must produce different hashes")
	}
}
