package kube

import (
	"crypto/sha256"
	"fmt"
)

func ComputeSHA256Sum(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
