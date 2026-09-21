package main

import (
	"errors"
	"fmt"
)

// "Pre-made flag for the known failure: not enough money.
//  Package-level so every function and caller shares the SAME value."
var ErrNoMoney = errors.New("not enough money")

// "withdraw takes balance and amount, returns (new balance, error).
//  If amount exceeds balance: balance unchanged + raise the ErrNoMoney flag.
//  Otherwise: new balance + nil (nil error = success)."
func withdraw(balance, amount int) (int, error) {
	if amount > balance {
		return balance, ErrNoMoney
	}
	return balance - amount, nil
}

// "payBill is a middle layer: it calls withdraw, and if that fails,
//  it wraps the error in an envelope marked 'payBill:' and passes it up.
//  %w = put the original error INSIDE the new one (keeps it reachable)."
func payBill(balance, amount int) (int, error) {
	newBalance, err := withdraw(balance, amount)
	if err != nil {
		return newBalance, fmt.Errorf("payBill: %w", err)
	}
	return newBalance, nil
}

func main() {
	// --- attempt 1: deliberately too much, forces the failure path ---
	newBalance, err := payBill(50, 100)
	if err != nil { // "did something go wrong?"
		// err is the ENVELOPE here, not the flag itself,
		// so == would fail. errors.Is opens the envelope.
		if errors.Is(err, ErrNoMoney) { // "is the flag inside ErrNoMoney?"
			fmt.Println("declined: not enough money")
		}
		fmt.Println("full story:", err) // prints: full story: payBill: not enough money
	} else {
		fmt.Println("new balance:", newBalance)
	}

	// --- attempt 2: valid amount, success path ---
	newBalance, err = payBill(50, 30) // = not :=, both variables already exist
	if err != nil {
		fmt.Println("unexpected:", err)
		return
	}
	fmt.Println("new balance:", newBalance) // new balance: 20
}



// expected output:

// declined: not enough money
// full story: payBill: not enough money
// new balance: 20
