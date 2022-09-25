package main

import "github.com/daystram/gambit/uci"

func runUCI() error {
	i := uci.NewInterface()
	return i.Run()
}
