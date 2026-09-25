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
