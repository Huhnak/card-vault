package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidCardNumber = errors.New("invalid card number")
	ErrCardExpired       = errors.New("card is expired")
	ErrInvalidCVV        = errors.New("invalid card cvv: must be 3 or 4 digits")
	ErrCardNotFound      = errors.New("card token not found")
)

type CardInput struct {
	CardNumber string `json:"card_number"`
	HolderName string `json:"holder_name"`
	ExpMonth   int    `json:"exp_month"`
	ExpYear    int    `json:"exp_year"`
	CVV        string `json:"cvv"`
}

func (c CardInput) String() string {
	return fmt.Sprintf("CardInput{Holder: %s, Exp: %02d/%04d, Pan: [MASKED], CVV: [REDACTED]}",
		c.HolderName, c.ExpMonth, c.ExpYear)
}

type TokenizedCard struct {
	ID        string    `json:"token"`
	MaskedPAN string    `json:"masked_pan"`
	ExpMonth  int       `json:"exp_month"`
	ExpYear   int       `json:"exp_year"`
	CreatedAt time.Time `json:"created_at"`
}
