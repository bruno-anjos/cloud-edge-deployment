package utils

import (
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
	EarthRadius = 6_378
)

func ChordAngleToKM(angle s1.ChordAngle) float64 {
	return angle.Angle().Radians() * EarthRadius
}
