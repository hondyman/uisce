package boresolver

// ValidateSelectedFields verifies that the provided field IDs exist on the BO definition.
// Returns a slice of invalid IDs (empty if all valid) or an error if the BO cannot be loaded.
func ValidateSelectedFields(repo BORepository, boID string, selected []string) ([]string, error) {
	if len(selected) == 0 {
		return nil, nil
	}

	boDef, err := repo.GetBODefinition(boID)
	if err != nil {
		return nil, err
	}

	valid := make(map[string]struct{}, len(boDef.Fields))
	for _, f := range boDef.Fields {
		valid[f.ID] = struct{}{}
	}

	invalid := make([]string, 0)
	for _, id := range selected {
		if _, ok := valid[id]; !ok {
			invalid = append(invalid, id)
		}
	}
	return invalid, nil
}
