package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

type View struct {
	Name        string                 `json:"name"`
	Extends     string                 `json:"extends,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Public      bool                   `json:"public,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Cubes       []ViewCube             `json:"cubes"`
	Dimensions  []Dimension            `json:"dimensions,omitempty"`
	Measures    []Measure              `json:"measures,omitempty"`
	Folders     []Folder               `json:"folders,omitempty"`
}

type ViewCube struct {
	JoinPath string   `json:"join_path"`
	Prefix   bool     `json:"prefix,omitempty"`
	Alias    string   `json:"alias,omitempty"`
	Includes []string `json:"includes"`
	Excludes []string `json:"excludes,omitempty"`
}

type Dimension struct {
	Name        string `json:"name"`
	Sql         string `json:"sql"`
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Public      bool   `json:"public,omitempty"`
}

type Measure struct {
	Name        string `json:"name"`
	Sql         string `json:"sql"`
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type Folder struct {
	Name     string       `json:"name"`
	Includes []FolderItem `json:"includes"`
}

type FolderItem struct {
	Name string `json:"name"`
}

func main() {
	// Get database connection from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	tenantID := os.Getenv("TENANT_ID")
	if tenantID == "" {
		tenantID = "default"
	}

	datasourceID := os.Getenv("DATASOURCE_ID")
	if datasourceID == "" {
		datasourceID = "default"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Seed views
	views := []View{
		{
			Name:        "base_customers",
			Title:       "Base Customers View",
			Description: "Basic customer information view for testing extends functionality",
			Public:      true,
			Cubes: []ViewCube{
				{
					JoinPath: "customers",
					Includes: []string{"*"},
				},
			},
			Dimensions: []Dimension{
				{
					Name:        "customer_id",
					Sql:         "${customers.id}",
					Type:        "number",
					Title:       "Customer ID",
					Description: "Unique identifier for customers",
					Public:      true,
				},
				{
					Name:        "customer_name",
					Sql:         "${customers.name}",
					Type:        "string",
					Title:       "Customer Name",
					Description: "Full name of the customer",
					Public:      true,
				},
			},
			Measures: []Measure{
				{
					Name:        "total_customers",
					Sql:         "COUNT(*)",
					Type:        "count",
					Title:       "Total Customers",
					Description: "Count of all customers",
				},
			},
		},
		{
			Name:        "extended_customers",
			Extends:     "base_customers",
			Title:       "Extended Customers View",
			Description: "Extended view that inherits from base_customers and adds more fields",
			Public:      true,
			Cubes: []ViewCube{
				{
					JoinPath: "customers.orders",
					Prefix:   true,
					Alias:    "orders",
					Includes: []string{"status", "amount"},
				},
			},
			Dimensions: []Dimension{
				{
					Name:        "order_status",
					Sql:         "${orders.status}",
					Type:        "string",
					Title:       "Order Status",
					Description: "Status of customer orders",
					Public:      true,
				},
			},
			Measures: []Measure{
				{
					Name:        "total_orders",
					Sql:         "COUNT(${orders.id})",
					Type:        "count",
					Title:       "Total Orders",
					Description: "Count of customer orders",
				},
				{
					Name:        "total_revenue",
					Sql:         "SUM(${orders.amount})",
					Type:        "sum",
					Title:       "Total Revenue",
					Description: "Sum of order amounts",
				},
			},
		},
		{
			Name:        "products_view",
			Title:       "Products View",
			Description: "View for product information",
			Public:      true,
			Cubes: []ViewCube{
				{
					JoinPath: "products",
					Includes: []string{"*"},
				},
			},
			Dimensions: []Dimension{
				{
					Name:        "product_id",
					Sql:         "${products.id}",
					Type:        "number",
					Title:       "Product ID",
					Description: "Unique identifier for products",
					Public:      true,
				},
				{
					Name:        "product_name",
					Sql:         "${products.name}",
					Type:        "string",
					Title:       "Product Name",
					Description: "Name of the product",
					Public:      true,
				},
			},
			Measures: []Measure{
				{
					Name:        "total_products",
					Sql:         "COUNT(*)",
					Type:        "count",
					Title:       "Total Products",
					Description: "Count of all products",
				},
			},
		},
	}

	for _, view := range views {
		if err := seedView(db, tenantID, datasourceID, view); err != nil {
			log.Printf("Failed to seed view %s: %v", view.Name, err)
		} else {
			log.Printf("Successfully seeded view: %s", view.Name)
		}
	}

	log.Println("View seeding completed")
}

func seedView(db *sql.DB, tenantID, datasourceID string, view View) error {
	viewJSON, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view JSON: %w", err)
	}

	// Use INSERT ... ON CONFLICT to handle both create and update
	_, err = db.Exec(`
		INSERT INTO public.views(tenant_id, tenant_datasource_id, name, view, created_by)
		VALUES ($1, $2, $3, $4, NULL)
		ON CONFLICT (tenant_id, tenant_datasource_id, name)
		DO UPDATE SET view = $4, updated_at = now()
	`, tenantID, datasourceID, view.Name, string(viewJSON))

	if err != nil {
		return fmt.Errorf("failed to insert/update view: %w", err)
	}

	return nil
}
