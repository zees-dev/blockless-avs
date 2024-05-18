package pkg

import (
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	avs "github.com/zees-dev/blockless-avs"
	proto "github.com/zees-dev/blockless-avs/node/proto"
)

// RegisterAPIRoutes sets up the API routes.
func RegisterAPIRoutes(cfg *avs.AppConfig, mux *http.ServeMux) {
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

	// newOracleUpdateChan
	mux.HandleFunc("POST /api/oracle", func(w http.ResponseWriter, r *http.Request) {
		// Parse the JSON body
		var req struct {
			Symbol string `json:"symbol"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Request an oracle update
		cfg.Operator.RequestOracleUpdate(req.Symbol)

		// Construct the response
		// Construct the response with full task details
		response := struct {
			Symbol    string `json:"symbol"`
			Timestamp uint32 `json:"timestamp"`
		}{
			Symbol: req.Symbol,
			// current timestmap now
			Timestamp: uint32(time.Now().Unix()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			cfg.Logger.Error("Failed to encode response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("POST /api/task", func(w http.ResponseWriter, r *http.Request) {
		// Parse the JSON body
		var req struct {
			Number string `json:"number"` // Use string to initially parse big integers
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Convert string to big.Int
		numToSquare := new(big.Int)
		if _, ok := numToSquare.SetString(req.Number, 10); !ok {
			cfg.Logger.Error("Invalid number format")
			http.Error(w, "Invalid number format", http.StatusBadRequest)
			return
		}

		// Send number to square to the task manager contract
		// NOTE: account assigned to operator pays for the gas
		newTask, taskIndex, err := cfg.Operator.SubmitNewTask(numToSquare)
		if err != nil {
			cfg.Logger.Error("Operator failed to send number to square", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Construct the response
		// Construct the response with full task details
		response := struct {
			NumberToBeSquared         string `json:"numberToBeSquared"`
			TaskCreatedBlock          uint32 `json:"taskCreatedBlock"`
			QuorumNumbers             []byte `json:"quorumNumbers"`
			QuorumThresholdPercentage uint32 `json:"quorumThresholdPercentage"`
			TaskIndex                 uint32 `json:"taskIndex"`
		}{
			NumberToBeSquared:         newTask.NumberToBeSquared.String(),
			TaskCreatedBlock:          newTask.TaskCreatedBlock,
			QuorumNumbers:             newTask.QuorumNumbers,
			QuorumThresholdPercentage: newTask.QuorumThresholdPercentage,
			TaskIndex:                 taskIndex,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			cfg.Logger.Error("Failed to encode response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
	})
}
