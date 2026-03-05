package password

import (
	"golang.org/x/crypto/bcrypt"
)

const cost = 12

// Hash hashes a password using bcrypt
func Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify compares a password with a hash
func Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
