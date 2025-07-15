package main

import (
	"fmt"
	"os"

	"github.com/beevik/ntp"
)

const myFormat = "2006-01-02 15:04:05 MST"

func main() {
	t, err := ntp.Time("pool.ntp.org")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting NTP time: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(t.Format(myFormat))
}
