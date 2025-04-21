package main

import (
	"context"
	"fmt"

	"go.leapkit.dev/tools/importmap/internal/importmap"
)

func main() {
	err := importmap.Process(context.Background())
	if err != nil {
		fmt.Println("[error]", err)
	}
}
