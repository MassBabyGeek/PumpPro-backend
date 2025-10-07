package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateUserID() string {
	return fmt.Sprintf("user_%d", time.Now().UnixNano()%1_000_000+int64(rand.Intn(999)))
}
