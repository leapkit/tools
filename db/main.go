package main

import (
	"fmt"
	"os"

	"go.leapkit.dev/tools/db/internal/database"
)

func main() {
	err := database.Exec()
	if err != nil {
		fmt.Printf("[error] %v\n", err)

		// Exit with error code 1 to signal failure
		// on requested operation.
		os.Exit(1)
	}
}
