// type Notifier interface { Notify(message string) error }
// Read as: "Define an interface called Notifier. Any type that has a Notify method taking a string and returning an error IS a Notifier." An interface is a contract - it says what a type must be able to do, not what it is.

// The big Go difference: there is no implements keyword. You never declare "EmailNotifier implements Notifier." If your type has the methods, it automatically satisfies the interface. This is called implicit implementation - and it's why Go code stays flexible.

// func alert(n Notifier, message string) error
// Read as: "alert takes any Notifier - email, SMS, Slack, anything invented next year - plus a message." This is polymorphism: one function, many behaviors.


package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// "Contract: to be a Notifier, you must have Notify(string) error"
type Notifier interface {
	Notify(message string) error
}

// "EmailNotifier has a Notify method, so it IS a Notifier (implicitly)"
type EmailNotifier struct {
	Address string
}

// Notify method bind by EmailNotifier type takes in a message of type string and returns an error
func (e EmailNotifier) Notify(message string) error {
	fmt.Printf("email to %s: %s\n", e.Address, message)
	return nil
}

type SMSNotifier struct {
	Phone string
}

func (s SMSNotifier) Notify(message string) error {
	fmt.Printf("sms to %s: %s\n", s.Phone, message)
	return nil
}

// "alert accepts ANY Notifier. It doesn't know or care which one."
func alert(n Notifier, message string) error {
	return n.Notify(message) // the RIGHT Notify runs based on the real type inside
}

// this is the reval interface behind every Go web serer.
type Handler interface {
	ServeHTTP(w io.Writer, message string)
}

func main() {
	// same function, two different behaviors: polymorphism
	alert(EmailNotifier{Address: "jaimin@example.com"}, "deploy done")
	alert(SMSNotifier{Phone: "+1-555-0100"}, "deploy done")

	// io.Writer: the most-used interface in Go's standard library.
	// os.Stdout IS a Writer. So is a file, a network connection, a buffer.
	// io.MultiWriter: one Write fans out to EVERY writer given.
	var buf bytes.Buffer // in-memory collection of bytes
	mw := io.MultiWriter(os.Stdout, &buf)
	mw.Write([]byte("log line\n")) // goes to terminal AND into buf
	fmt.Printf("buffer captured: %q\n", buf.String())
}
