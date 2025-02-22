package main

import (
	"fmt"

	"go.leapkit.dev/tools/db/internal/database"
)

func main() {
	err := database.Exec()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
	}
}
