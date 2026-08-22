package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to alpha database.")

	rows, err := db.Query(`
		SELECT id, tenant_id, key, name, is_core, fields, config
		FROM business_objects
	`)
	if err != nil {
		log.Fatalf("Failed to query business_objects: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var boID, tenantID, key, name string
		var isCore bool
		var fieldsJSON, configJSON []byte

		if err := rows.Scan(&boID, &tenantID, &key, &name, &isCore, &fieldsJSON, &configJSON); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		log.Printf("Processing BO: %s (%s, tenant=%s, isCore=%v)", name, key, tenantID, isCore)

		type BOFieldItem struct {
			ID             string `json:"id"`
			Key            string `json:"key"`
			Name           string `json:"name"`
			DisplayName    string `json:"displayName"`
			TechnicalName  string `json:"technicalName"`
			DataType       string `json:"dataType"`
			Type           string `json:"type"`
			SemanticTermID string `json:"semanticTermId"`
			Description    string `json:"description"`
			Sequence       int    `json:"sequence"`
		}

		var fieldList []BOFieldItem
		if len(fieldsJSON) > 0 && string(fieldsJSON) != "[]" && string(fieldsJSON) != "null" {
			_ = json.Unmarshal(fieldsJSON, &fieldList)
		}

		if len(fieldList) == 0 && len(configJSON) > 0 {
			var cfg map[string]interface{}
			if err := json.Unmarshal(configJSON, &cfg); err == nil {
				if fRaw, ok := cfg["fields"]; ok {
					fBytes, _ := json.Marshal(fRaw)
					_ = json.Unmarshal(fBytes, &fieldList)
				}
			}
		}

		if len(fieldList) == 0 {
			edgeRows, err := db.Query(`
				SELECT cn.id, cn.node_name, cn.properties
				FROM catalog_edge ce
				JOIN catalog_node cn ON ce.target_node_id = cn.id
				JOIN catalog_edge_types cet ON ce.edge_type_id = cet.id
				WHERE ce.source_node_id = $1::uuid AND cet.edge_type_name = 'HAS_FIELD'
			`, boID)
			if err == nil {
				seq := 1
				for edgeRows.Next() {
					var fID, fName string
					var fPropsJSON []byte
					if err := edgeRows.Scan(&fID, &fName, &fPropsJSON); err == nil {
						var props map[string]interface{}
						_ = json.Unmarshal(fPropsJSON, &props)

						stID, _ := props["semantic_term_node_id"].(string)
						fItem := BOFieldItem{
							ID:             fID,
							Key:            fName,
							Name:           fName,
							DisplayName:    fName,
							TechnicalName:  fName,
							DataType:       "number",
							Type:           "number",
							SemanticTermID: stID,
							Sequence:       seq,
						}
						fieldList = append(fieldList, fItem)
						seq++
					}
				}
				edgeRows.Close()
			}
		}

		log.Printf("  Found %d fields to sync into bo_fields", len(fieldList))

		for idx, f := range fieldList {
			fKey := f.Key
			if fKey == "" {
				fKey = f.Name
			}
			fName := f.DisplayName
			if fName == "" {
				fName = f.Name
			}
			fDisp := f.DisplayName
			if fDisp == "" {
				fDisp = fName
			}
			fTech := f.TechnicalName
			if fTech == "" {
				fTech = fKey
			}
			fType := f.Type
			if fType == "" {
				fType = f.DataType
			}
			if fType == "" {
				fType = "number"
			}
			seq := f.Sequence
			if seq == 0 {
				seq = idx + 1
			}

			fieldUUID := f.ID
			if _, err := uuid.Parse(fieldUUID); err != nil || fieldUUID == "" {
				ns := uuid.NameSpaceURL
				fieldUUID = uuid.NewSHA1(ns, []byte(fmt.Sprintf("bofield:%s:%s:%s", tenantID, key, fKey))).String()
			}

			_, err := db.Exec(`
				INSERT INTO bo_fields (
					id, tenant_id, business_object_id, key, name, display_name,
					technical_name, type, is_core, sequence, created_at, last_modified_at
				) VALUES (
					$1::uuid, $2::uuid, $3::uuid, $4, $5, $6,
					$7, $8, $9, $10, NOW(), NOW()
				)
				ON CONFLICT (id) DO UPDATE SET
					name = EXCLUDED.name,
					display_name = EXCLUDED.display_name,
					technical_name = EXCLUDED.technical_name,
					type = EXCLUDED.type,
					is_core = EXCLUDED.is_core,
					sequence = EXCLUDED.sequence,
					last_modified_at = NOW()
			`, fieldUUID, tenantID, boID, fKey, fName, fDisp, fTech, fType, isCore, seq)

			if err != nil {
				log.Printf("    Error upserting bo_field %s: %v", fKey, err)
			} else {
				log.Printf("    ✓ Synced bo_field: %s (%s)", fDisp, fieldUUID)
			}
		}

		if len(fieldList) > 0 {
			var cfg map[string]interface{}
			if len(configJSON) > 0 {
				_ = json.Unmarshal(configJSON, &cfg)
			}
			if cfg == nil {
				cfg = make(map[string]interface{})
			}
			cfg["fields"] = fieldList
			newCfgJSON, _ := json.Marshal(cfg)

			_, _ = db.Exec(`
				UPDATE business_objects
				SET config = $1
				WHERE id = $2::uuid
			`, newCfgJSON, boID)
		}
	}

	log.Println("\n✓ Synchronized all bo_fields successfully!")
}
