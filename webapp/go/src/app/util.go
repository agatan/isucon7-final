package main

import (
	"log"
	"math/big"
	"strconv"
)

var (
	bi1000 = big.NewInt(1000)
)

func str2big(s string) *big.Int {
	x := new(big.Int)
	x.SetString(s, 10)
	return x
}

func big2exp(n *big.Int) Exponential {
	s := n.String()

	if len(s) <= 15 {
		return Exponential{n.Int64(), 0}
	}

	t, err := strconv.ParseInt(s[:15], 10, 64)
	if err != nil {
		log.Panic(err)
	}
	return Exponential{t, int64(len(s) - 15)}
}
