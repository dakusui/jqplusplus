package nodelevel

import "fmt"

// FetchParentNodeMock is an example function to simulate retrieving JSON objects by ID.
func FetchParentNodeMock(id string) (map[string]interface{}, error) {
	// Simulated JSON data for parent nodes by ID.
	parentData := map[string]map[string]interface{}{
		"parent1": {"name": "Parent 1", "config": map[string]interface{}{"enabled": true}},
		"parent2": {"name": "Parent 2", "config": map[string]interface{}{"enabled": false, "mode": "test"}},
	}

	if data, ok := parentData[id]; ok {
		return data, nil
	}

	return nil, fmt.Errorf("node with ID '%s' not found", id)
}
