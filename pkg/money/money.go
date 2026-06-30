package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/bojanz/currency"
)

const UzsCode = "UZS"

type Money struct {
	amount currency.Amount
}

func FromSums(amount string) (Money, error) {
	a, err := currency.NewAmount(amount, UzsCode)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: a,
	}, nil
}

func FromTiyins(amount int) (Money, error) {
	return FromTiyins64(int64(amount))
}

func FromTiyins64(amount int64) (Money, error) {
	a, err := currency.NewAmountFromInt64(amount, UzsCode)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: a,
	}, nil
}

func Zero() Money {
	m, err := FromTiyins(0)
	if err != nil {
		panic(err)
	}

	return m
}

func (a *Money) Add(b Money) (Money, error) {
	result, err := a.amount.Add(b.amount)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: result,
	}, nil
}

func (a *Money) Sub(b Money) (Money, error) {
	result, err := a.amount.Sub(b.amount)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: result,
	}, nil
}

func (a *Money) Mul(n string) (Money, error) {
	result, err := a.amount.Mul(n)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: result,
	}, nil
}

func (a *Money) Div(n string) (Money, error) {
	result, err := a.amount.Div(n)
	if err != nil {
		return Money{}, err
	}

	return Money{
		amount: result,
	}, nil
}

func (a *Money) Cmp(b Money) int {
	cmp, err := a.amount.Cmp(b.amount)
	if err != nil {
		panic(err)
	}

	return cmp
}

func (a *Money) Equal(b Money) bool {
	return a.Cmp(b) == 0
}

func (a *Money) LessThan(b Money) bool {
	return a.Cmp(b) < 0
}

func (a *Money) LessThanOrEqual(b Money) bool {
	return a.Cmp(b) <= 0
}

func (a *Money) GreaterThan(b Money) bool {
	return a.Cmp(b) > 0
}

func (a *Money) GreaterThanOrEqual(b Money) bool {
	return a.Cmp(b) >= 0
}

func (a *Money) IsPositive() bool {
	return a.amount.IsPositive()
}

func (a *Money) IsNegative() bool {
	return a.amount.IsNegative()
}

func (a *Money) IsZero() bool {
	return a.amount.IsZero()
}

func (a *Money) GetTiyins() int64 {
	val, err := a.amount.Int64()
	if err != nil {
		panic(err)
	}

	return val
}

func (a *Money) GetSums() string {
	return a.amount.Number()
}

func (a *Money) MarshalJSON() ([]byte, error) {
	value := a.GetSums()
	if value == "-0.00" {
		value = "0.00"
	}

	return json.Marshal(value)
}

func (a *Money) UnmarshalJSON(data []byte) error {
	var value string

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	m, err := FromSums(value)
	if err != nil {
		return err
	}

	*a = m

	return nil
}

func (a *Money) Value() (driver.Value, error) {
	return a.GetTiyins(), nil
}

func (a *Money) Scan(src interface{}) error {
	amount, ok := src.(int64)
	if !ok {
		return fmt.Errorf("failed to scan Money from type:%T", src)
	}

	m, err := FromTiyins64(amount)
	if err != nil {
		return fmt.Errorf("failed to scan Money: %s", err.Error())
	}
	*a = m

	return nil
}
