package utils

import (
	"crypto/rand"
	"math/big"
)

func GetRandInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}

	return int(n.Int64())
}
