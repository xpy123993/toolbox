package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ShellTask specifies the structure to execute a shell command.
type ShellTask struct {
	Base      string   `json:"base"`
	Arguments []string `json:"args"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: inserter http://.../insert command args...")
		return
	}
	shellTask := ShellTask{
		Base:      os.Args[2],
		Arguments: os.Args[3:],
	}
	data, err := json.Marshal(shellTask)
	if err != nil {
		fmt.Printf("cannot wrap request: %v\n", err)
		return
	}
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%s?task=%s", os.Args[1], data))
	if err != nil {
		fmt.Printf("cannot commit the task: %v\n", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Printf("task master returns error status: %d\n", resp.StatusCode)
		return
	}

	if data, err := io.ReadAll(resp.Body); err == nil {
		fmt.Printf("task is successfully commited as `%s`\n", data)
	}
}
