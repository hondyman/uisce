package main

import (
	"fmt"
)

func main() {
	db, err := e2eDB()
	if err != nil {
		fmt.Println("error connecting to database:", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT workflow_id FROM temporal_workflows")
	if err != nil {
		fmt.Println("error querying temporal_workflows:", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var workflowID string
		if err := rows.Scan(&workflowID); err != nil {
			fmt.Println("scan error:", err)
			return
		}
		count++
	}
	if count == 0 {
		fmt.Println("no workflows found")
		return
	}
	fmt.Println("found workflows:", count)
}
