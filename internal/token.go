package internal

import (
	"fmt"
	"math/rand"
)

func GenToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
