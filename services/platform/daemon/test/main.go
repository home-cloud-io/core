package main

import (
	"fmt"
	"slices"
)

func main() {

	s := []string{"10.100.0.2/32", "10.100.0.3/32", "10.100.0.5/32"}

	for i := 2; i < 255; i++ {
		address := fmt.Sprintf("10.100.0.%d/32", i)
		fmt.Println(address)
		if !slices.Contains(s, address) {
			break
		}
	}
}
