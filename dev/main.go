package main

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"go.leapkit.dev/tools/dev/internal/rebuilder"
)

func main() {
	pflag.Parse()

	err := rebuilder.Serve(context.Background())
	if err != nil {
		fmt.Println("[error] starting the server:", err)
	}
}
