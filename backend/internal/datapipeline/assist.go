package datapipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// --- platform catalog --------------------------------------------------------

// BOInfo is a business object an analyst can read or write.
type BOInfo struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// RuleInfo is a catalog validation rule.
type RuleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description,omitempty"`
}

// PlatformCatalog is what exists for the requesting tenant. Implementations
// are request-scoped: the tenant is already fixed.
type PlatformCatalog interface {
	BusinessObjects(ctx context.Context) ([]BOInfo, error)
	BOFields(ctx context.Context, boKey string) ([]TargetField, error)
	Rules(ctx context.Context, boKey string) ([]RuleInfo, error)
	Files(ctx context.Context) ([]string, error)
	StagingTables(ctx context.Context) ([]StagingTable, error)
}

// Issue is a problem on a node (NodeID empty: the whole pipeline).
type Issue struct {
	NodeID  string `json:"node_id,omitempty"`
	Message string `json:"message"`
}

// Check is structural validation plus grounding: every business object,
// field, rule, file and staging table the spec names must exist for the
// tenant. It is what both the editor and the assistant rely on, so a pipeline
// can never refer to something that is not there.
func Check(ctx context.Context, cat PlatformCatalog, spec *Spec) []Issue {
	var out []Issue
	for _, e := range spec.Validate() {
		out = append(out, issueFromError(e))
	}
	if cat == nil {
		return out
	}
	add := func(node, f string, a ...any) { out = append(out, Issue{NodeID: node, Message: fmt.Sprintf(f, a...)}) }

	bos, err := cat.BusinessObjects(ctx)
	boKnown := map[string]bool{}
	for _, b := range bos {
		boKnown[b.Key] = true
	}
	fieldCache := map[string]map[string]bool{}
	fieldsOf := func(key string) map[string]bool {
		if f, ok := fieldCache[key]; ok {
			return f
		}
		fs, ferr := cat.BOFields(ctx, key)
		m := map[string]bool{}
		for _, f := range fs {
			m[f.Name] = true
		}
		if ferr != nil {
			m = nil
		}
		fieldCache[key] = m
		return m
	}
	checkBO := func(node, key string) bool {
		if key == "" || err != nil {
			return false
		}
		if !boKnown[key] {
			add(node, "there is no business object %q", key)
			return false
		}
		return true
	}

	var files map[string]bool
	var tables map[string]map[string]bool
	for _, n := range spec.Nodes {
		switch n.Type {
		case NodeBOSource:
			var c BOSourceConfig
			if json.Unmarshal(n.Config, &c) == nil && checkBO(n.ID, c.BOKey) {
				if f := fieldsOf(c.BOKey); f != nil {
					for _, cond := range c.Filters {
						if cond.Field != "" && !f[cond.Field] {
							add(n.ID, "%s has no field %q", c.BOKey, cond.Field)
						}
					}
				}
			}
		case NodeBOSink:
			var c BOSinkConfig
			if json.Unmarshal(n.Config, &c) == nil && checkBO(n.ID, c.BOKey) {
				if f := fieldsOf(c.BOKey); f != nil {
					for _, k := range c.KeyFields {
						if !f[k] {
							add(n.ID, "%s has no field %q to match on", c.BOKey, k)
						}
					}
				}
			}
		case NodeRuleCheck:
			var c struct {
				RuleIDs []string `json:"rule_ids"`
				BOKey   string   `json:"bo_key"`
			}
			if json.Unmarshal(n.Config, &c) != nil || len(c.RuleIDs) == 0 {
				continue
			}
			if c.BOKey == "" {
				add(n.ID, "say which business object's rules to apply")
				continue
			}
			if !checkBO(n.ID, c.BOKey) {
				continue
			}
			rules, rerr := cat.Rules(ctx, c.BOKey)
			if rerr != nil {
				continue
			}
			known := map[string]bool{}
			for _, r := range rules {
				known[r.ID] = true
			}
			for _, id := range c.RuleIDs {
				if !known[id] {
					add(n.ID, "rule %q is not an active rule of %s", id, c.BOKey)
				}
			}
		case NodeFileSource:
			var c FileSourceConfig
			if json.Unmarshal(n.Config, &c) != nil || c.URI == "" {
				continue
			}
			if files == nil {
				list, ferr := cat.Files(ctx)
				if ferr != nil {
					continue
				}
				files = map[string]bool{}
				for _, f := range list {
					files[f] = true
				}
			}
			if !files[strings.TrimPrefix(strings.TrimPrefix(c.URI, "file://"), "/")] {
				add(n.ID, "file %q has not been uploaded", c.URI)
			}
		case NodeStagingSink:
			var c StagingSinkConfig
			if json.Unmarshal(n.Config, &c) != nil || c.Table == "" {
				continue
			}
			if tables == nil {
				list, terr := cat.StagingTables(ctx)
				if terr != nil {
					continue
				}
				tables = map[string]map[string]bool{}
				for _, t := range list {
					cols := map[string]bool{}
					for _, col := range t.Columns {
						cols[col.Name] = true
					}
					tables[t.Table] = cols
				}
			}
			cols, ok := tables[c.Table]
			if !ok {
				add(n.ID, "there is no loadable staging table %s", c.Table)
				continue
			}
			for field, col := range c.Columns {
				if !cols[col] {
					add(n.ID, "%s: %s has no column %q", field, c.Table, col)
				}
			}
		}
	}

	// A map feeding a business object must produce that object's fields.
	for _, n := range spec.Nodes {
		if n.Type != NodeMap {
			continue
		}
		var c MapConfig
		if json.Unmarshal(n.Config, &c) != nil {
			continue
		}
		sink := downstreamSinkOf(spec, n.ID)
		if sink == nil || sink.Type != NodeBOSink {
			continue
		}
		var sc BOSinkConfig
		if json.Unmarshal(sink.Config, &sc) != nil || !boKnown[sc.BOKey] {
			continue
		}
		f := fieldsOf(sc.BOKey)
		if f == nil {
			continue
		}
		for _, m := range c.Fields {
			if m.To != "" && !f[m.To] {
				add(n.ID, "%s has no field %q (mapped from %s)", sc.BOKey, m.To, m.From)
			}
		}
	}
	return out
}

func downstreamSinkOf(spec *Spec, id string) *Node {
	kids := map[string][]string{}
	for _, e := range spec.Edges {
		kids[e.From] = append(kids[e.From], e.To)
	}
	byID := map[string]*Node{}
	for i := range spec.Nodes {
		byID[spec.Nodes[i].ID] = &spec.Nodes[i]
	}
	queue, seen := append([]string(nil), kids[id]...), map[string]bool{}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if seen[cur] || byID[cur] == nil {
			continue
		}
		seen[cur] = true
		switch byID[cur].Type {
		case NodeBOSink, NodeStagingSink, NodeFileSink:
			return byID[cur]
		case NodeMap:
		default:
			queue = append(queue, kids[cur]...)
		}
	}
	return nil
}

func issueFromError(e error) Issue {
	msg := e.Error()
	if strings.HasPrefix(msg, `node "`) {
		if end := strings.Index(msg[6:], `"`); end >= 0 {
			return Issue{NodeID: msg[6 : 6+end], Message: strings.TrimPrefix(msg[6+end+1:], ": ")}
		}
	}
	return Issue{Message: msg}
}

// --- assistant ---------------------------------------------------------------

// LLM completes a prompt.
type LLM func(ctx context.Context, prompt string) (string, error)

// ChatTurn is one earlier message in the conversation.
type ChatTurn struct {
	Role string `json:"role"` // user | assistant
	Text string `json:"text"`
}

// AssistRequest asks the assistant to answer or change the pipeline.
type AssistRequest struct {
	Message        string     `json:"message"`
	Spec           Spec       `json:"spec"`
	SelectedNodeID string     `json:"selected_node_id,omitempty"`
	History        []ChatTurn `json:"history,omitempty"`
	// PreviewRejects lets the assistant explain what went wrong in a preview.
	PreviewRejects []string `json:"preview_rejects,omitempty"`
}

// AssistResponse is the assistant's answer. Spec is a proposed replacement
// (nil when it only answered); the editor shows it for the analyst to apply.
type AssistResponse struct {
	Reply    string   `json:"reply"`
	Spec     *Spec    `json:"spec,omitempty"`
	Changes  []string `json:"changes,omitempty"`
	Issues   []Issue  `json:"issues,omitempty"` // problems the proposal still has
	Attempts int      `json:"attempts"`
}

// Assistant drafts and edits pipelines from plain language, grounded in the
// tenant's platform (business objects, fields, rules, files, staging tables)
// and checked by Check before the analyst ever sees a proposal.
type Assistant struct {
	LLM         LLM
	MaxAttempts int
}

const assistMaxContextBOs = 8

func (a *Assistant) Respond(ctx context.Context, cat PlatformCatalog, req AssistRequest) (*AssistResponse, error) {
	if a.LLM == nil {
		return nil, fmt.Errorf("the AI assistant is not configured for this environment")
	}
	if strings.TrimSpace(req.Message) == "" {
		return nil, fmt.Errorf("message is required")
	}
	if req.Spec.Version == 0 {
		req.Spec.Version = SpecVersion
	}
	pctx, err := buildContext(ctx, cat, req)
	if err != nil {
		return nil, err
	}
	max := a.MaxAttempts
	if max <= 0 {
		max = 3
	}

	prompt := assistPrompt(pctx, req)
	var best *AssistResponse
	for attempt := 1; attempt <= max; attempt++ {
		raw, err := a.LLM(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("the AI service failed: %w", err)
		}
		out, perr := parseAssistant(raw)
		if perr != nil {
			prompt = assistPrompt(pctx, req) + "\n\nYour previous answer was not valid JSON (" + perr.Error() + "). Answer again with only the JSON object."
			continue
		}
		out.Attempts = attempt
		if out.Spec == nil {
			return out, nil // an answer, no change
		}
		if out.Spec.Version == 0 {
			out.Spec.Version = SpecVersion
		}
		keepPositions(out.Spec, &req.Spec)
		out.Issues = Check(ctx, cat, out.Spec)
		if len(out.Issues) == 0 {
			return out, nil
		}
		best = out
		var b strings.Builder
		b.WriteString("\n\nYour proposed pipeline has these problems. Fix them using ONLY names from the context, then answer again:\n")
		for _, is := range out.Issues {
			if is.NodeID != "" {
				fmt.Fprintf(&b, "- step %s: %s\n", is.NodeID, is.Message)
			} else {
				fmt.Fprintf(&b, "- %s\n", is.Message)
			}
		}
		specJSON, _ := json.Marshal(out.Spec)
		prompt = assistPrompt(pctx, req) + "\n\nYour previous proposal:\n" + string(specJSON) + b.String()
	}
	if best == nil {
		return nil, fmt.Errorf("the AI service did not return a usable answer; try rephrasing")
	}
	return best, nil
}

// keepPositions carries canvas positions over for nodes that still exist and
// lays out new ones after them, so an edit does not scramble the canvas.
func keepPositions(next, prev *Spec) {
	pos := map[string]*Position{}
	maxX := 0.0
	for _, n := range prev.Nodes {
		if n.Position != nil {
			pos[n.ID] = n.Position
			if n.Position.X > maxX {
				maxX = n.Position.X
			}
		}
	}
	order, _ := next.TopoOrder()
	idx := map[string]int{}
	for i, id := range order {
		idx[id] = i
	}
	for i := range next.Nodes {
		n := &next.Nodes[i]
		if p, ok := pos[n.ID]; ok {
			n.Position = p
		} else if n.Position == nil {
			n.Position = &Position{X: 60 + float64(idx[n.ID])*280, Y: 120}
		}
	}
}

type promptContext struct {
	BusinessObjects []BOInfo                 `json:"business_objects"`
	BOFields        map[string][]TargetField `json:"business_object_fields"`
	Rules           map[string][]RuleInfo    `json:"rules"`
	Files           []string                 `json:"uploaded_files"`
	StagingTables   []StagingTable           `json:"staging_tables"`
}

// buildContext gathers what the model may name. Field and rule detail is
// included for the business objects the spec uses or the message mentions.
func buildContext(ctx context.Context, cat PlatformCatalog, req AssistRequest) (*promptContext, error) {
	pc := &promptContext{BOFields: map[string][]TargetField{}, Rules: map[string][]RuleInfo{}}
	var err error
	if pc.BusinessObjects, err = cat.BusinessObjects(ctx); err != nil {
		return nil, fmt.Errorf("listing business objects: %w", err)
	}
	pc.Files, _ = cat.Files(ctx)
	pc.StagingTables, _ = cat.StagingTables(ctx)

	relevant := map[string]int{}
	msg := strings.ToLower(req.Message)
	for _, b := range pc.BusinessObjects {
		score := 0
		for _, w := range []string{strings.ToLower(b.Key), strings.ToLower(b.Label)} {
			if w != "" && strings.Contains(msg, w) {
				score += 2
			}
			for _, tok := range strings.FieldsFunc(w, func(r rune) bool { return r == '_' || r == ' ' }) {
				if len(tok) > 3 && strings.Contains(msg, tok) {
					score++
				}
			}
		}
		if score > 0 {
			relevant[b.Key] = score
		}
	}
	for _, n := range req.Spec.Nodes {
		var c struct {
			BOKey string `json:"bo_key"`
		}
		if json.Unmarshal(n.Config, &c) == nil && c.BOKey != "" {
			relevant[c.BOKey] += 10
		}
	}
	keys := make([]string, 0, len(relevant))
	for k := range relevant {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if relevant[keys[i]] != relevant[keys[j]] {
			return relevant[keys[i]] > relevant[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > assistMaxContextBOs {
		keys = keys[:assistMaxContextBOs]
	}
	for _, k := range keys {
		if f, ferr := cat.BOFields(ctx, k); ferr == nil {
			pc.BOFields[k] = f
		}
		if r, rerr := cat.Rules(ctx, k); rerr == nil && len(r) > 0 {
			pc.Rules[k] = r
		}
	}
	return pc, nil
}

const assistInstructions = `You help a data analyst build a data pipeline in the uisce platform. You never write code; you edit a pipeline document.

A pipeline is JSON: {"version":1,"nodes":[...],"edges":[{"from":"<id>","to":"<id>"}],"error_policy":"skip_and_log"}.
Each node: {"id":"<short_snake_id>","type":"<type>","label":"<plain words>","config":{...}}. Each non-source node has exactly one input edge. Sources have none; destinations have no outgoing edges.

Node types and config:
- file_source: {"uri":"<an uploaded file path>","format":"csv|json|parquet","delimiter":",|\||;|tab","has_header":true,"columns":[{"name":"<file column>","type":"string|int|float|decimal|bool|date|timestamp","nullable":false}]}. Leave columns [] if you do not know the file's columns; the analyst will read the file.
- bo_source: {"bo_key":"<business object key>","filters":[{"field":"<field>","operator":"<op>","value":...}],"limit":0}. Operators: equals, not_equals, greater_than, greater_equal, less_than, less_equal, between ([low,high]), in / not_in (list), contains, starts_with, ends_with, before, after, is_null, is_not_null, is_true, is_false.
- validate: {"required":["<field>"],"unique":["<field>"]}
- rule_check: {"bo_key":"<business object whose rules>","rule_ids":["<rule id>"]}. BLOCK rules reject rows; WARN rules only warn.
- map: {"fields":[{"from":"<incoming field>","to":"<outgoing field>","transform":"trim|upper|lower|to_date|to_number|lookup","lookup":{"A":"Alpha"}}],"keep_unmapped":false}. Before a business object destination, "to" must be that object's field names.
- bo_sink: {"bo_key":"<business object key>","mode":"create|upsert","key_fields":["<field>"],"dry_run":false}. Every record is checked by the object's rules.
- staging_sink: {"table":"staging.<name>","source_cd":"<SOURCE>","domain":"<DOMAIN>","run_ref":"<optional load reference>","columns":{"<field>":"<column>"}}
- file_sink: {"uri":"exports/<name>.<ext>","format":"csv|json|parquet"}

Rules for you:
- Use ONLY business objects, fields, rule ids, files and staging tables listed in the context. Never invent one. If something needed is missing, say so in reply and leave it out.
- Keep existing node ids when you change a node. Change only what the analyst asked for.
- When the analyst only asks a question, answer it and set "spec" to null.

Answer with ONLY a JSON object: {"reply":"<short plain-language answer or summary of what you did>","changes":["<one line per change>"],"spec":<the whole updated pipeline, or null>}`

func assistPrompt(pc *promptContext, req AssistRequest) string {
	ctxJSON, _ := json.Marshal(pc)
	specJSON, _ := json.Marshal(stripPositions(req.Spec))
	var b strings.Builder
	b.WriteString(assistInstructions)
	b.WriteString("\n\nContext (what exists for this analyst):\n")
	b.Write(ctxJSON)
	b.WriteString("\n\nCurrent pipeline:\n")
	b.Write(specJSON)
	if req.SelectedNodeID != "" {
		fmt.Fprintf(&b, "\n\nThe analyst has step %q selected.", req.SelectedNodeID)
	}
	if len(req.PreviewRejects) > 0 {
		b.WriteString("\n\nThe last preview rejected rows for these reasons:\n")
		for i, r := range req.PreviewRejects {
			if i >= 30 {
				break
			}
			b.WriteString("- " + r + "\n")
		}
	}
	if len(req.History) > 0 {
		b.WriteString("\n\nConversation so far:\n")
		start := 0
		if len(req.History) > 10 {
			start = len(req.History) - 10
		}
		for _, t := range req.History[start:] {
			fmt.Fprintf(&b, "%s: %s\n", t.Role, t.Text)
		}
	}
	b.WriteString("\n\nAnalyst: " + req.Message)
	return b.String()
}

func stripPositions(s Spec) Spec {
	out := s
	out.Nodes = make([]Node, len(s.Nodes))
	for i, n := range s.Nodes {
		n.Position = nil
		out.Nodes[i] = n
	}
	return out
}

// parseAssistant extracts the JSON object from a model answer (tolerating a
// code fence or leading prose).
func parseAssistant(raw string) (*AssistResponse, error) {
	s := strings.TrimSpace(raw)
	start, end := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object found")
	}
	var out struct {
		Reply   string          `json:"reply"`
		Changes []string        `json:"changes"`
		Spec    json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &out); err != nil {
		return nil, err
	}
	resp := &AssistResponse{Reply: out.Reply, Changes: out.Changes}
	if len(out.Spec) > 0 && string(out.Spec) != "null" {
		var sp Spec
		if err := json.Unmarshal(out.Spec, &sp); err != nil {
			return nil, fmt.Errorf("spec: %w", err)
		}
		resp.Spec = &sp
	}
	if resp.Reply == "" && resp.Spec == nil {
		return nil, fmt.Errorf("empty answer")
	}
	return resp, nil
}
