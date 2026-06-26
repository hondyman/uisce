package codegen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hondyman/semlayer/backend/internal/apistudio"
)

const tsClientTemplate = `
import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';

export interface ClientConfig {
  baseURL: string;
  token?: string;
}

{{range .Endpoints}}
export interface {{.Name}}Params {
  {{range .Params}}
  {{.Name}}: {{.Type}};
  {{end}}
}

export interface {{.Name}}Response {
  [key: string]: any;
}
{{end}}

export class SemanticClient {
  private client: AxiosInstance;

  constructor(config: ClientConfig) {
    this.client = axios.create({
      baseURL: config.baseURL,
      headers: config.token ? { Authorization: "Bearer " + config.token } : {},
    });
  }

  {{range .Endpoints}}
  /**
   * {{.Description}}
   */
  async {{.MethodName}}(params: {{.Name}}Params): Promise<{{.Name}}Response[]> {
    const resp = await this.client.get('{{.Path}}', { params });
    return resp.data;
  }
  {{end}}
}
`

type EndpointMeta struct {
	Name        string
	MethodName  string
	Path        string
	Description string
	Params      []ParamMeta
}

type ParamMeta struct {
	Name string
	Type string
}

func GenerateTypeScript(inputPath, outDir string) error {
	// Read endpoints
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	var endpoints []apistudio.APIEndpoint
	if err := json.Unmarshal(data, &endpoints); err != nil {
		return err
	}

	// Prepare metadata
	var meta []EndpointMeta
	for _, ep := range endpoints {
		meta = append(meta, EndpointMeta{
			Name:        capitalize(ep.Name),
			MethodName:  lowerFirst(ep.Name),
			Path:        ep.Path,
			Description: "",                      // Not available in Core APIEndpoint
			Params:      parseParams(ep.Filters), // naive param parsing
		})
	}

	// Create output dir
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Generate file
	tmpl, err := template.New("ts-client").Parse(tsClientTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(outDir, "client.ts"))
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]interface{}{
		"Endpoints": meta,
	})
}

func capitalize(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerFirst(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func parseParams(paramsJSON json.RawMessage) []ParamMeta {
	// Assuming params is a map[string]interface{}
	var params map[string]interface{}
	json.Unmarshal(paramsJSON, &params)

	var res []ParamMeta
	for k := range params {
		// Guess type or default to string
		res = append(res, ParamMeta{Name: k, Type: "string"})
	}
	return res
}
