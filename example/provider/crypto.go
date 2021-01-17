package provider

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Crypto provider for secure operations
type Crypto struct {
	Config CryptoConfig `di:"config"`
}

// CryptoConfig is a configuration of secrets
type CryptoConfig interface {
	GetPasswordHashCost() int
}

// HashPassword make bcrypt hash from password
func (pvd *Crypto) HashPassword(password string) (string, error) {
	res, err := bcrypt.GenerateFromPassword([]byte(password), pvd.Config.GetPasswordHashCost())
	return string(res), fmt.Errorf("hashing password: %w", err)
}

// CheckPassword compare hash and password and return true if equal
func (*Crypto) CheckPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, fmt.Errorf("check password: %w", err)
}
