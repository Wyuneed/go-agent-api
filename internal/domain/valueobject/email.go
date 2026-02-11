package valueobject

import (
	"fmt"
	"net/mail"
	"strings"
)

type Email struct {
	value string
}

func NewEmail(address string) (Email, error) {
	address = strings.TrimSpace(strings.ToLower(address))
	if address == "" {
		return Email{}, fmt.Errorf("email cannot be empty")
	}
	if _, err := mail.ParseAddress(address); err != nil {
		return Email{}, fmt.Errorf("invalid email format: %s", address)
	}
	return Email{value: address}, nil
}

func (e Email) String() string {
	return e.value
}

func (e Email) Domain() string {
	parts := strings.SplitN(e.value, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func (e Email) Equals(other Email) bool {
	return e.value == other.value
}
