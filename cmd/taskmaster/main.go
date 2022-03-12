package main

import (
	"fmt"
	"os"

	"github.com/xpy123993/toolbox/cmd/taskmaster/cmd"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Usage: taskmaster [serve | work | insert] [args]")
		return
	}
	switch os.Args[1] {
	case "serve":
		if err := cmd.HandleServe(os.Args[2:]...); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case "work":
		if err := cmd.HandleWorker(os.Args[2:]...); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case "insert":
		if err := cmd.HandleInsert(os.Args[2:]...); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	default:
		fmt.Println("Usage: taskmaster [serve | work | insert] [args]")
		os.Exit(1)
	}
}
