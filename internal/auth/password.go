package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword — хэширует пароль с помощью bcrypt (DefaultCost).
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword — сравнивает пароль с хэшем, возвращает true при совпадении.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
