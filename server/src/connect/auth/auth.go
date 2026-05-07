package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"aidanwoods.dev/go-paseto"
)

const tokenTTL = 24 * time.Hour

type Auth struct {
	mu        sync.Mutex
	nonces    map[string]time.Time
	nonceTTL  time.Duration
	secretKey paseto.V4AsymmetricSecretKey
	publicKey paseto.V4AsymmetricPublicKey
}

func parsePASERK(prefix, paserk string) ([]byte, error) {
	if !strings.HasPrefix(paserk, prefix) {
		return nil, fmt.Errorf("expected key with prefix %q", prefix)
	}
	return base64.RawURLEncoding.DecodeString(strings.TrimPrefix(paserk, prefix))
}

func NewAuth(privatePASERK, publicPASERK string, nonceTTL time.Duration) (*Auth, error) {
	skBytes, err := parsePASERK("k4.secret.", privatePASERK)
	if err != nil {
		return nil, err
	}
	sk, err := paseto.NewV4AsymmetricSecretKeyFromBytes(skBytes)
	if err != nil {
		return nil, err
	}
	pkBytes, err := parsePASERK("k4.public.", publicPASERK)
	if err != nil {
		return nil, err
	}
	pk, err := paseto.NewV4AsymmetricPublicKeyFromBytes(pkBytes)
	if err != nil {
		return nil, err
	}
	return &Auth{
		nonces:    make(map[string]time.Time),
		nonceTTL:  nonceTTL,
		secretKey: sk,
		publicKey: pk,
	}, nil
}

func (a *Auth) IssueNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(b)
	a.mu.Lock()
	a.nonces[nonce] = time.Now().Add(a.nonceTTL)
	a.mu.Unlock()
	return nonce, nil
}

func (a *Auth) ConsumeNonce(passKey string) bool {
	parser := paseto.NewParser()
	token, err := parser.ParseV4Public(a.publicKey, passKey, nil)
	if err != nil {
		return false
	}
	raw, err := token.GetString("nonce")
	if err != nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	exp, ok := a.nonces[raw]
	if !ok || time.Now().After(exp) {
		return false
	}
	delete(a.nonces, raw)
	return true
}

func (a *Auth) NewToken() string {
	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(tokenTTL))
	return token.V4Sign(a.secretKey, nil)
}

func (a *Auth) VerifyToken(tokenStr string) error {
	parser := paseto.NewParser()
	_, err := parser.ParseV4Public(a.publicKey, tokenStr, nil)
	return err
}
