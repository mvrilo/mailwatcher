package main

import (
	"os"

	mail "github.com/mvrilo/mailwatcher"
)

func main() {
	g, err := mail.New(os.Getenv("EMAIL"), os.Getenv("PASS"), os.Getenv("ADDR"))
	if err != nil {
		panic(err)
	}

	println("[#] Waiting...\n")
	g.WatchFunc(2, func(email mail.Message) {
		println("[+] got new mail!")
		print("from: ", string(email.Header.Get("from")), "\n")
		print("subject: ", string(email.Header.Get("subject")), "\n")
	})
}
