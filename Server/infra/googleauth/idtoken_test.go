package googleauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestCompleteRejectsInvalidIdentity(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(map[string]any){
		"nonce":                func(c map[string]any) { c["nonce"] = "wrong" },
		"issuer":               func(c map[string]any) { c["iss"] = "https://attacker.example" },
		"audience":             func(c map[string]any) { c["aud"] = "other-client" },
		"expiration":           func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() },
		"subject":              func(c map[string]any) { c["sub"] = "" },
		"authorized party":     func(c map[string]any) { c["azp"] = "other-client" },
		"access token binding": func(c map[string]any) { c["at_hash"] = "incorrect" },
		"multiple audiences without authorized party": func(c map[string]any) { c["aud"] = []string{"client-id", "other-client"} },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newProviderFixture(t)
			a := f.begin(t)
			payload := claims(a.Nonce)
			mutate(payload)
			f.idToken = f.sign(t, payload)
			if _, err := f.client.Complete(context.Background(), a, a.State, "valid-code"); err == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
}

func TestCompleteRejectsInvalidSignature(t *testing.T) {
	t.Parallel()

	f := newProviderFixture(t)
	a := f.begin(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f.key = otherKey
	f.idToken = f.sign(t, claims(a.Nonce))
	if _, err := f.client.Complete(context.Background(), a, a.State, "valid-code"); err == nil {
		t.Fatal("invalid signature accepted")
	}
}
