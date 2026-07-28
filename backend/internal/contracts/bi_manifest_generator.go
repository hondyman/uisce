package contracts

import (
	"fmt"
	"strings"
)

// BIConnectorManifestGenerator builds BI-specific metadata connection manifests (.pbids, .tds, Cube.js schema)
type BIConnectorManifestGenerator struct{}

func NewBIConnectorManifestGenerator() *BIConnectorManifestGenerator {
	return &BIConnectorManifestGenerator{}
}

// GeneratePowerBIManifest (.pbids) for 1-click PowerBI connection setup
func (g *BIConnectorManifestGenerator) GeneratePowerBIManifest(host string, port int, dbName string) string {
	return fmt.Sprintf(`{
  "version": "0.1",
  "connections": [
    {
      "details": {
        "protocol": "postgresql",
        "address": {
          "server": "%s",
          "port": "%d",
          "database": "%s"
        }
      },
      "options": {
        "DirectQuery": true
      },
      "mode": "DirectQuery"
    }
  ]
}`, host, port, dbName)
}

// GenerateTableauDataSourcesManifest (.tds) with semantic folders & definitions
func (g *BIConnectorManifestGenerator) GenerateTableauDataSourcesManifest(host string, port int, dbName string, boName string, fields []string) string {
	var colsXML strings.Builder
	for _, f := range fields {
		colsXML.WriteString(fmt.Sprintf(`      <column name='[%s]' datatype='string' role='dimension' type='nominal' />%s`, f, "\n"))
	}

	return fmt.Sprintf(`<?xml version='1.0' encoding='utf-8' ?>
<datasource formatted-name='UisceSemanticOS' inline='true' source-platform='mac' version='18.1'>
  <connection class='postgres' dbname='%s' server='%s' port='%d' username='postgres'>
    <relation name='%s' table='[%s]' type='table' />
%s  </connection>
</datasource>`, dbName, host, port, boName, boName, colsXML.String())
}

// GenerateCubeDevSchema generates Cube.js semantic model definitions dynamically
func (g *BIConnectorManifestGenerator) GenerateCubeDevSchema(boName string, dimensions []string, measures []string) string {
	var dimsJS strings.Builder
	for _, d := range dimensions {
		dimsJS.WriteString(fmt.Sprintf("    %s: {\n      sql: `%s`,\n      type: `string`\n    },\n", d, d))
	}

	var measJS strings.Builder
	for _, m := range measures {
		measJS.WriteString(fmt.Sprintf("    total_%s: {\n      sql: `%s`,\n      type: `sum`\n    },\n", m, m))
	}

	return fmt.Sprintf(`cube('%s', {
  sql: 'SELECT * FROM "%s"',

  measures: {
%s  },

  dimensions: {
%s  }
});`, boName, boName, measJS.String(), dimsJS.String())
}
