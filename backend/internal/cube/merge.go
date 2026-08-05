package cube

func MergeCube(base Cube, ext Cube) (Cube, []ValidationIssue) {
	return base, nil
}

func ValidateExtension(base Cube, ext Cube) []ValidationIssue {
	return nil
}

func ComposeCatalog(coreCubes map[string]Cube, extCubes map[string]Cube) (map[string]Cube, []ValidationIssue) {
	result := make(map[string]Cube)
	for k, v := range coreCubes {
		result[k] = v
	}
	for k, v := range extCubes {
		result[k] = v
	}
	return result, nil
}
