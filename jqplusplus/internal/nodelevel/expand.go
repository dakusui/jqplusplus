package nodelevel

import (
	"errors"
	"fmt"
)

// ExpandNodeLevelInheritances handles the "extends" property of a node and merges associated objects.
// nodeContent: JSON content of the current node.
// fetchParentNode: Function that retrieves the JSON content of a parent node by its identifier.
func ExpandNodeLevelInheritances(nodeContent map[string]interface{}, fetchParentNode func(string) (map[string]interface{}, error)) (map[string]interface{}, error) {
	currentContent := nodeContent

	// Handle the "$extends" property for normal filelevel
	if extends, ok := nodeContent["$extends"].([]interface{}); ok {
		for _, parentNodeID := range extends {
			parentStr, ok := parentNodeID.(string)
			if !ok {
				return nil, errors.New("invalid parent node reference")
			}

			// Fetch parent content using the provided function.
			parentContent, err := fetchParentNode(parentStr)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve parent node '%s': %w", parentStr, err)
			}

			// Merge parent content into the current content.
			currentContent, err = mergeObjects(parentContent, currentContent)
			if err != nil {
				return nil, fmt.Errorf("failed to merge parent node '%s': %w", parentStr, err)
			}
		}
	}

	return currentContent, nil
}

// mergeObjects merges two JSON objects, giving priority to properties in object2 in case of conflicts.
func mergeObjects(obj1, obj2 map[string]interface{}) (map[string]interface{}, error) {
	merged := make(map[string]interface{})

	// Copy all key-value pairs from obj1 into merged.
	for key, value := range obj1 {
		merged[key] = value
	}

	// Override or add all key-value pairs from obj2 into merged.
	for key, value := range obj2 {
		switch v := value.(type) {
		case map[string]interface{}:
			// If both values are objects, merge them recursively.
			if v1, ok := merged[key].(map[string]interface{}); ok {
				mergedObj, err := mergeObjects(v1, v)
				if err != nil {
					return nil, fmt.Errorf("failed to merge nested objects for key '%s': %w", key, err)
				}
				merged[key] = mergedObj
			} else {
				merged[key] = v
			}
		default:
			// For other types, obj2's value will overwrite obj1's value.
			merged[key] = value
		}
	}

	return merged, nil
}
