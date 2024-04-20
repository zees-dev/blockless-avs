package pkg

import (
	"encoding/json"
	"net/http"

	"github.com/zees-dev/blockless-avs/core"
	proto "github.com/zees-dev/blockless-avs/node/proto"
)

// RegisterAPIRoutes sets up the API routes.
func RegisterAPIRoutes(cfg *core.AppConfig, mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Example handler that marshals a protobuf message to JSON and writes it to the response
	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		// Create an instance of the protobuf message
		appMeta := &proto.AppMeta{
			Name: cfg.AppName,
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
	})

	// Register the handler function for the route
	mux.HandleFunc("GET /api/meta", func(w http.ResponseWriter, r *http.Request) {

		// Note: Consider error handling for production code
		jsonData, err := json.Marshal(proto.API{
			EndPoints: &proto.API_End_Points{
				GetMetaData: "/api/getMeta",
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Write the JSON data to the response
		w.Write(jsonData)
	})
}
