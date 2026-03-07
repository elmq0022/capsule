package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	run()
}

func run() {
	if len(os.Args) < 3 {
		fmt.Println("did not provide a program to run")
		return
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Printf(
			"running command %s returned error: %q\n",
			os.Args[2],
			err,
		)
		os.Exit(1)
	}
}
