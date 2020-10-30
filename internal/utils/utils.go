package utils

import (
	"math"
	"math/rand"

	"github.com/golang/geo/s1"
)

func RandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

const (
	earthDiameter = 6_378 * 2
)

func ChordAngleToKM(angle s1.ChordAngle) float64 {
	return math.Sqrt(float64(angle)) * earthDiameter
}
