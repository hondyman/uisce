package dynamic

import (
	"github.com/hondyman/uisce/backend/internal/cube"
)

type CubeDynamicEnhancer struct {
	baseCube *cube.Cube
}

func NewCubeDynamicEnhancer(baseCube *cube.Cube) *CubeDynamicEnhancer {
	return &CubeDynamicEnhancer{
		baseCube: baseCube,
	}
}

func (cde *CubeDynamicEnhancer) GenerateCubeJSConfig(params []DynamicParameter, dynamicMeasures []DynamicMeasure) (map[string]any, error) {
	return map[string]any{}, nil
}

func (cde *CubeDynamicEnhancer) GenerateParameterSchema(params []DynamicParameter) (map[string]any, error) {
	return map[string]any{}, nil
}
