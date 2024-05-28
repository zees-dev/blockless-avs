package pkg

import (
	"sync"

	b7sAPI "github.com/blocklessnetwork/b7s/api"
)

// ExecuteResponse describes the REST API response for function execution.
type ExecuteFunctionResponse struct {
	b7sAPI.ExecuteResponse
	FunctionID string `json:"function_id"`
}

func NewExecuteFunctionResponse(functionID string, res b7sAPI.ExecuteResponse) ExecuteFunctionResponse {
	return ExecuteFunctionResponse{
		ExecuteResponse: res,
		FunctionID:      functionID,
	}
}

// FunctionRegistry holds registered functions and their subscribers.
type FunctionRegistry struct {
	mu          sync.RWMutex
	subscribers map[string]bool
	ch          *chan ExecuteFunctionResponse
}

// NewFunctionRegistry creates a new FunctionRegistry.
func NewFunctionRegistry(ch *chan ExecuteFunctionResponse) *FunctionRegistry {
	return &FunctionRegistry{
		subscribers: make(map[string]bool),
		ch:          ch,
	}
}

// RegisterFunction registers one or more function IDs in the registry.
func (fr *FunctionRegistry) RegisterFunctions(functionIDs ...string) {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	for _, functionID := range functionIDs {
		if _, exists := fr.subscribers[functionID]; !exists {
			fr.subscribers[functionID] = true
		}
	}
}

// Notify notifies all subscribers of the function result.
func (fr *FunctionRegistry) Notify(functionID string, result ExecuteFunctionResponse) {
	fr.mu.RLock()
	defer fr.mu.RUnlock()
	if _, exists := fr.subscribers[functionID]; exists {
		*fr.ch <- result
	}
}
