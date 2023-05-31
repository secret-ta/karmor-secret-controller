package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"log"
)

func main() {
	http.HandleFunc("/demo", demoHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func demoHandler(w http.ResponseWriter, r *http.Request) {
	cmd := r.URL.Query().Get("cmd")

	// Run the command specified by the cmd query parameter and capture its output
	// Warning: the security of this operation is not guaranteed
	command := exec.Command(cmd)
	command.Env = []string{} // Set environment to an empty slice
	out, err := command.Output()
	if err != nil {
		log.Printf("Error starting command %s: %s\n", cmd, err)
		http.Error(w, fmt.Sprintf("Failed to start command: %s", cmd), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Command %s started successfully. Output:\n%s", cmd, out)
}