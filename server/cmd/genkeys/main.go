package main

import (
	"encoding/base64"
	"fmt"

	"aidanwoods.dev/go-paseto"
)

func main() {
	sk := paseto.NewV4AsymmetricSecretKey()
	fmt.Println("PRIVATE_PASETO_KEY=k4.secret." + base64.RawURLEncoding.EncodeToString(sk.ExportBytes()))
	fmt.Println("PUBLIC_PASETO_KEY=k4.public." + base64.RawURLEncoding.EncodeToString(sk.Public().ExportBytes()))
}
