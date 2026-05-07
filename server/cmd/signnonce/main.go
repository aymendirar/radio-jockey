package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: signnonce <nonce>")
		os.Exit(1)
	}
	nonce := os.Args[1]

	paserk := os.Getenv("PRIVATE_PASETO_KEY")
	if paserk == "" {
		fmt.Fprintln(os.Stderr, "PRIVATE_PASETO_KEY not set")
		os.Exit(1)
	}

	skBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(paserk, "k4.secret."))
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid key:", err)
		os.Exit(1)
	}
	sk, err := paseto.NewV4AsymmetricSecretKeyFromBytes(skBytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid key:", err)
		os.Exit(1)
	}

	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(2 * time.Minute))
	token.SetString("nonce", nonce)

	fmt.Println(token.V4Sign(sk, nil))
}
