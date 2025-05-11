package util

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

var rng *rand.Rand

func init() {
	seed := uint64(time.Now().UnixNano())
	stream := uint64(time.Now().UnixNano() >> 1) // or any other entropy source
	rng = rand.New(rand.NewPCG(seed, stream))
}

// RandomInt generates a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + rng.Int64N(max-min+1)
}

// RandomString generates a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rng.IntN(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomOwner generates a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// RandomMoney generates a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// RandomCurrency generates a random currency code
func RandomCurrency() string {
	currencies := []string{USD, EUR, CAD}
	n := len(currencies)
	return currencies[rng.IntN(n)]
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@email.com", RandomString(6))
}
