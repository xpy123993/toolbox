package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	serverFlags = flag.NewFlagSet("server", flag.PanicOnError)
)

func handleServerMode(args ...string) {
	address := serverFlags.String("address", ":8080", "Serving address.")
	baseCommand := serverFlags.String("base", "", "The path of the interpreter to execute commands.")
	hashedPasswordStr := serverFlags.String("password", "", "The token generated from token command.")
	serverFlags.Parse(args)

	hashedPassword, err := base64.RawStdEncoding.DecodeString(*hashedPasswordStr)
	if err != nil {
		log.Fatalf("failed to parse hash password flag: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(r.Header.Get("X-CHANNEL-TOKEN"))); err != nil {
			http.Error(w, fmt.Sprintf("error: %v", err), http.StatusUnauthorized)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cmd := exec.CommandContext(r.Context(), *baseCommand, serverFlags.Args()...)
		for k, v := range r.Form {
			if len(v) == 0 {
				continue
			}
			if !strings.HasPrefix(k, "WEBHOOK_ENV_") {
				http.Error(w, "Environment variable must has prefix `WEBHOOK_ENV_`.", http.StatusBadRequest)
				return
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v[0]))
		}
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	log.Printf("Serving at %s", *address)
	http.ListenAndServe(*address, nil)
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("Usage: webhook_server [token|serve] args...")
		return
	}
	switch strings.ToLower(os.Args[1]) {
	case "token":
		if len(os.Args) == 2 {
			fmt.Println("Password is missing.")
			os.Exit(1)
		}
		data, err := bcrypt.GenerateFromPassword([]byte(os.Args[2]), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error: %v", err)
			os.Exit(1)
		}
		fmt.Println(base64.RawStdEncoding.EncodeToString(data))
	case "serve":
		handleServerMode(os.Args[2:]...)
	default:
		fmt.Println("Usage: webhook_server [token|serve] args...")
		os.Exit(1)
	}
}
