// This package creates a new leapkit module using the gonew tool
// and the leapkit/template template. It passes the name
// of the module as an argument to the gonew tool.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

// new tool creates a new leapkit module using the gonew tool
// and the go.leapkit.dev/template template. It receives the name of
// the newly created module as an argument.
func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: new [name]")
		return
	}

	// new tool invokes gonew under the hood to create a
	// new module using the go.leapkit.dev/template template.
	cmd := exec.Command(
		"go", "run", "rsc.io/tmp/gonew@latest",
		"go.leapkit.dev/template@latest",
		args[1],
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
