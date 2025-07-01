package main

import (
	"fmt"
	"net/http"
)

// healthcheck handler
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "status: available")
	fmt.Fprintf(w, "environment: %s\n", app.config.Env)
}
