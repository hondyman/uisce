package main

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type TestService struct{}

func (s *TestService) Echo(msg string) string {
	return "Echo: " + msg
}

func TestInspectServiceBindings(t *testing.T) {
	svc := application.NewService(&TestService{})
	app := application.New(application.Options{
		Name: "TestApp",
		Services: []application.Service{
			svc,
		},
	})
	fmt.Printf("App initialized with service: %v\n", app != nil)

	// Inspect service methods and type
	typ := reflect.TypeOf(svc)
	for i := 0; i < typ.NumMethod(); i++ {
		fmt.Printf("Service method: %s\n", typ.Method(i).Name)
	}
}
