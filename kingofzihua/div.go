package main

import (
	"errors"
	"fmt"
)

func div(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}

	return a / b, nil
}

func main() {
	fmt.Println(div(1, 1))
	fmt.Println(div(1, 0))
	fmt.Println(div(1, -1))
}
