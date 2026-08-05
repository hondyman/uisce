package analytics

import "github.com/hondyman/uisce/backend/models"

func MergeCube(base models.Cube, ext models.Cube) (models.Cube, []models.ValidationIssue) {
	return base, nil
}

func ValidateExtension(base models.Cube, ext models.Cube) []models.ValidationIssue {
	return nil
}

func ComposeCatalog(coreCubes map[string]models.Cube, extCubes map[string]models.Cube) (map[string]models.Cube, []models.ValidationIssue) {
	result := make(map[string]models.Cube)
	for k, v := range coreCubes {
		result[k] = v
	}
	for k, v := range extCubes {
		result[k] = v
	}
	return result, nil
}
