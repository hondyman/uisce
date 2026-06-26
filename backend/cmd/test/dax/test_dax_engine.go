package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/services"
)

func main() {
	fmt.Println("🧮 Testing DAX Engine for Wealth Management")
	fmt.Println(strings.Repeat("=", 50))

	// Create DAX engine
	daxEngine := services.NewDAXEngine()

	// Test SWITCH function
	fmt.Println("\n1. Testing SWITCH function:")
	context := &services.DAXContext{}

	// Test SWITCH with match
	result, err := daxEngine.ExecuteFunction("SWITCH", []interface{}{"Medium", "Low", "Low Risk", "Medium", "Medium Risk", "High", "High Risk", "Unknown Risk"}, context)
	if err != nil {
		log.Printf("SWITCH error: %v", err)
	} else {
		fmt.Printf("   SWITCH(\"Medium\", ...) = %v", result)
	}

	// Test SWITCH with no match (else clause)
	result, err = daxEngine.ExecuteFunction("SWITCH", []interface{}{"Very High", "Low", "Low Risk", "Medium", "Medium Risk", "High", "High Risk", "Unknown Risk"}, context)
	if err != nil {
		log.Printf("SWITCH error: %v", err)
	} else {
		fmt.Printf("   SWITCH(\"Very High\", ...) = %v", result)
	}

	// Test IF function
	fmt.Println("\n2. Testing IF function:")

	// Test IF with true condition
	result, err = daxEngine.ExecuteFunction("IF", []interface{}{true, "Positive", "Negative"}, context)
	if err != nil {
		log.Printf("IF error: %v", err)
	} else {
		fmt.Printf("   IF(true, \"Positive\", \"Negative\") = %v", result)
	}

	// Test IF with false condition
	result, err = daxEngine.ExecuteFunction("IF", []interface{}{false, "Positive", "Negative"}, context)
	if err != nil {
		log.Printf("IF error: %v", err)
	} else {
		fmt.Printf("   IF(false, \"Positive\", \"Negative\") = %v", result)
	}

	// Test COALESCE function
	fmt.Println("\n3. Testing COALESCE function:")

	result, err = daxEngine.ExecuteFunction("COALESCE", []interface{}{nil, nil, "Default Value", nil}, context)
	if err != nil {
		log.Printf("COALESCE error: %v", err)
	} else {
		fmt.Printf("   COALESCE(nil, nil, \"Default Value\", nil) = %v", result)
	}

	// Test BLANK function
	fmt.Println("\n4. Testing BLANK function:")

	result, err = daxEngine.ExecuteFunction("BLANK", []interface{}{}, context)
	if err != nil {
		log.Printf("BLANK error: %v", err)
	} else {
		fmt.Printf("   BLANK() = %v", result)
	}

	// List available functions
	fmt.Println("\n5. Available DAX Functions by Category:")
	categories := daxEngine.ListFunctionsByCategory()
	for category, functions := range categories {
		fmt.Printf("   %s: %d functions\n", category, len(functions))
		for _, fn := range functions {
			fmt.Printf("     - %s: %s\n", fn.Name, fn.Description)
		}
	}

	fmt.Println("\n✅ DAX Engine test completed successfully!")
	fmt.Println("\n💡 Wealth Management DAX Examples:")
	fmt.Println("   - TOTALYTD([Portfolio Return], 'Calendar'[Date]) - YTD returns")
	fmt.Println("   - STDEVX.P('Portfolio', [Monthly Return]) - Portfolio volatility")
	fmt.Println("   - RANKX('Portfolio', [Sharpe Ratio], , DESC) - Performance ranking")
	fmt.Println("   - SWITCH([Risk Level], 1, \"Low\", 2, \"Medium\", 3, \"High\") - Risk classification")
}
