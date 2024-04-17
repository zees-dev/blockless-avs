package main

import (
	"encoding/json"
	"net/http"

	go_vite_app "github.com/dmikey/go-vite-app-p2p/server/proto"
)

var myAPI = &go_vite_app.API{
	EndPoints: &go_vite_app.API_End_Points{
		GetMetaData: "/api/getMeta",
	},
}

// RegisterAPIRoutes sets up the API routes.
func RegisterAPIRoutes(cfg Cfg) {
	// Example handler that marshals a protobuf message to JSON and writes it to the response
	getAppMeta := func(w http.ResponseWriter, r *http.Request) {
		// Create an instance of the protobuf message
		appMeta := &go_vite_app.AppMeta{
			Name: cfg.appname,
		}

		// Convert protobuf message to JSON
		// Note: Consider error handling for production code
		jsonData, err := json.Marshal(appMeta)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Write the JSON data to the response
		w.Write(jsonData)
	}

	getAPIMeta := func(w http.ResponseWriter, r *http.Request) {

		// Note: Consider error handling for production code
		jsonData, err := json.Marshal(myAPI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Write the JSON data to the response
		w.Write(jsonData)
	}

	http.HandleFunc("/api", getAPIMeta)

	// Register the handler function for the route
	http.HandleFunc(myAPI.EndPoints.GetMetaData, getAppMeta)
}
