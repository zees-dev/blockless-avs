package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	b7sAPI "github.com/blocklessnetwork/b7s/api"
	b7sNode "github.com/blocklessnetwork/b7s/node"

	"github.com/blocklessnetwork/b7s/models/blockless"
	"github.com/blocklessnetwork/b7s/models/execute"
	"github.com/blocklessnetwork/b7s/node/aggregate"
	avs "github.com/zees-dev/blockless-avs"
)

// RegisterAPIRoutes sets up the API routes.
func RegisterAPIRoutes(node *b7sNode.Node, cfg *avs.CoreConfig, mux *http.ServeMux, registry *FunctionRegistry) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /functions/install", func(w http.ResponseWriter, r *http.Request) {
		var req b7sAPI.InstallFunctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.URI == "" && req.CID == "" {
			http.Error(w, "URI or CID are required", http.StatusBadRequest)
			return
		}

		// Add a deadline to the context.
		const functionInstallTimeout = 10 * time.Second
		reqCtx, cancel := context.WithTimeout(r.Context(), functionInstallTimeout)
		defer cancel()

		// Start function install in a separate goroutine and signal when it's done.
		fnErr := make(chan error)
		go func() {
			err := node.PublishFunctionInstall(reqCtx, req.URI, req.CID, req.Subgroup)
			fnErr <- err
		}()

		// Wait until either function install finishes, or request times out.
		select {
		// Context timed out.
		case <-reqCtx.Done():
			status := http.StatusRequestTimeout
			if !errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
				status = http.StatusInternalServerError
			}
			http.Error(w, "Function installation timed out", status)
			return

		// Work done.
		case err := <-fnErr:
			cfg.Logger.Error("Failed to install function", "err", err)
			if err != nil {
				http.Error(w, "Function installation failed", http.StatusInternalServerError)
			}
			break
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Register a new function ID
	// TODO: this should be a protected function which AVS operators can call
	mux.HandleFunc("POST /functions/register", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FunctionID string `json:"function_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.FunctionID == "" {
			http.Error(w, "Function ID is required", http.StatusBadRequest)
			return
		}

		registry.RegisterFunctions(req.FunctionID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Function ID registered"))
	})

	mux.HandleFunc("POST /functions/execute", func(w http.ResponseWriter, r *http.Request) {
		var req b7sAPI.ExecuteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if the function ID is registered.
		if _, exists := registry.subscribers[req.FunctionID]; !exists {
			http.Error(w, "Function ID not registered", http.StatusNotFound)
			return
		}

		// Get the execution result.
		exr := execute.Request{
			Config:     req.Config,
			FunctionID: req.FunctionID,
			Method:     req.Method,
			Parameters: req.Parameters,
		}
		code, id, results, cluster, err := node.ExecuteFunction(r.Context(), exr, req.Topic)
		if err != nil {
			cfg.Logger.Warn("node failed to execute function", "function", req.FunctionID, "err", err)
		}

		// Transform the node response format to the one returned by the API.
		res := b7sAPI.ExecuteResponse{
			Code:      code,
			RequestID: id,
			Results:   aggregate.Aggregate(results),
			Cluster:   cluster,
		}

		// Communicate the reason for failure in these cases.
		if errors.Is(err, blockless.ErrRollCallTimeout) || errors.Is(err, blockless.ErrExecutionNotEnoughNodes) {
			res.Message = err.Error()
		}

		// Send the response.
		jsonData, err := json.Marshal(res)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Notify subscribers.
		go registry.Notify(req.FunctionID, NewExecuteFunctionResponse(req.FunctionID, res))

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	})

	mux.HandleFunc("POST /functions/requests/result", func(w http.ResponseWriter, r *http.Request) {
		var req b7sAPI.ExecutionResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			cfg.Logger.Error("Failed to decode JSON request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.ID == "" {
			http.Error(w, "missing request ID", http.StatusBadRequest)
			return
		}

		// Lookup execution result.
		result, ok := node.ExecutionResult(req.ID)
		if !ok {
			http.Error(w, "Execution result not found", http.StatusNotFound)
		}

		// Send the response back.
		jsonData, err := json.Marshal(result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	})

	// Register the handler function for the route
	mux.HandleFunc("GET /meta", func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			PeerID  string `json:"peer_id"`
			P2PPort uint32 `json:"p2p_port"`
		}{
			PeerID:  node.ID(),
			P2PPort: uint32(cfg.BlocklessConfig.Connectivity.Port),
		}

		// Set content type to JSON for the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			cfg.Logger.Error("Failed to encode response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
		}
	})
}
