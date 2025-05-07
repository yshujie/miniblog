package auth

import "golang.org/x/crypto/bcrypt"

// Encrypt 加密密码
func Encrypt(source string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(source), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// Compare 比较密码
func Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
