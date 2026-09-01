package apistudio

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/cbo"
	"github.com/hondyman/uisce/backend/internal/region"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// ODataHandler exposes a read-only OData v4 surface over the same
// Studio-published REST GET endpoints APIRuntime serves — so Excel /
// Power Query (and any other OData-aware client) can consume Business
// Objects natively, through the exact same entitlement stack (role gate,
// row filter, field masking) and tenant-scoped execution as REST/GraphQL.
//
// Scope, deliberately limited for v1:
//   - Read-only (GET). No $filter beyond "field eq value" clauses ANDed
//     together — anything else (gt/lt/contains/or/nested groups/...) is
//     rejected with 400 rather than silently ignored, since silently
//     dropping a filter the client thinks scoped the result set is a data
//     exposure risk, not a convenience.
//   - One entity set per Business Object, resolved from that BO's first
//     active "rest"/"GET" endpoint definition. A BO published through
//     multiple GET endpoints picks one arbitrarily — documented, not fixed,
//     since API Studio's endpoint model doesn't have a canonical "the OData
//     entity set for this BO" concept yet.
//   - $metadata's entity Key is "id" if present in the endpoint's field
//     list, else the first field — APIEndpoint doesn't track a real primary
//     key column, so this is a best-effort guess, not authoritative.
type ODataHandler struct {
	repo        *Repository
	resolver    *analytics.BOContextResolver
	db          *sqlx.DB
	planCache   *GraphQLPlanCache
	rateLimiter *RateLimiter
	mountPrefix string
}

// NewODataHandler creates a new OData handler.
func NewODataHandler(repo *Repository, resolver *analytics.BOContextResolver, db *sqlx.DB, redisClient *redis.Client) *ODataHandler {
	return &ODataHandler{
		repo:        repo,
		resolver:    resolver,
		db:          db,
		planCache:   NewGraphQLPlanCache(redisClient),
		rateLimiter: NewRateLimiter(redisClient),
	}
}

// SetMountPrefix records the path prefix this handler is mounted under
// (e.g. "/odata"), mirroring APIRuntime.SetMountPrefix.
func (h *ODataHandler) SetMountPrefix(prefix string) {
	h.mountPrefix = prefix
}

// ServeHTTP dispatches the service document, $metadata, or an entity-set
// query based on the path remainder after the mount prefix.
func (h *ODataHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, h.mountPrefix)
	remainder = strings.Trim(remainder, "/")

	env := r.Header.Get("X-Env")
	if env == "" {
		env = "production"
	}
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID

	switch remainder {
	case "":
		h.serveServiceDocument(w, r, env, tenantIDStr)
	case "$metadata":
		h.serveMetadata(w, r, env, tenantIDStr)
	default:
		h.serveEntitySet(w, r, remainder, env, tenantIDStr)
	}
}

// boEntitySet is one Business Object exposed as an OData entity set,
// resolved from its Studio-published GET endpoint.
type boEntitySet struct {
	Name   string
	BOName string
	Fields []string
	Ep     *APIEndpoint
}

func (h *ODataHandler) listEntitySets(r *http.Request, env, tenantID string) ([]boEntitySet, error) {
	eps, err := h.repo.ListEndpoints(r.Context(), env, tenantID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var sets []boEntitySet
	for i := range eps {
		ep := &eps[i]
		if ep.Type != "rest" || strings.ToUpper(ep.Method) != "GET" || ep.Status != "active" {
			continue
		}
		if seen[ep.BOName] {
			continue
		}
		seen[ep.BOName] = true

		fields := decodeStringList(ep.Fields)
		sets = append(sets, boEntitySet{Name: ep.BOName, BOName: ep.BOName, Fields: fields, Ep: ep})
	}
	return sets, nil
}

func (h *ODataHandler) serveServiceDocument(w http.ResponseWriter, r *http.Request, env, tenantID string) {
	sets, err := h.listEntitySets(r, env, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing entity sets: %v", err), http.StatusInternalServerError)
		return
	}

	type entry struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
		URL  string `json:"url"`
	}
	values := make([]entry, 0, len(sets))
	for _, s := range sets {
		values = append(values, entry{Name: s.Name, Kind: "EntitySet", URL: s.Name})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"@odata.context": h.mountPrefix + "/$metadata",
		"value":          values,
	})
}

// Minimal CSDL structures — just enough to describe string-typed entity
// properties and one key, which is all APIEndpoint's field metadata carries.
type edmxRoot struct {
	XMLName xml.Name `xml:"edmx:Edmx"`
	Xmlns   string   `xml:"xmlns:edmx,attr"`
	Version string   `xml:"Version,attr"`
	Data    edmxData `xml:"edmx:DataServices"`
}
type edmxData struct {
	Schema edmSchema `xml:"Schema"`
}
type edmSchema struct {
	Xmlns     string          `xml:"xmlns,attr"`
	Namespace string          `xml:"Namespace,attr"`
	Entities  []edmEntityType `xml:"EntityType"`
	Container edmContainer    `xml:"EntityContainer"`
}
type edmEntityType struct {
	Name       string        `xml:"Name,attr"`
	Key        *edmKey       `xml:"Key"`
	Properties []edmProperty `xml:"Property"`
}
type edmKey struct {
	PropertyRef edmPropertyRef `xml:"PropertyRef"`
}
type edmPropertyRef struct {
	Name string `xml:"Name,attr"`
}
type edmProperty struct {
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
}
type edmContainer struct {
	Name       string         `xml:"Name,attr"`
	EntitySets []edmEntitySet `xml:"EntitySet"`
}
type edmEntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

func (h *ODataHandler) serveMetadata(w http.ResponseWriter, r *http.Request, env, tenantID string) {
	sets, err := h.listEntitySets(r, env, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing entity sets: %v", err), http.StatusInternalServerError)
		return
	}

	const namespace = "ApiStudio"
	schema := edmSchema{
		Xmlns:     "http://docs.oasis-open.org/odata/ns/edm",
		Namespace: namespace,
		Container: edmContainer{Name: "Container"},
	}

	for _, s := range sets {
		keyField := ""
		for _, f := range s.Fields {
			if f == "id" {
				keyField = f
				break
			}
		}
		if keyField == "" && len(s.Fields) > 0 {
			keyField = s.Fields[0]
		}

		entity := edmEntityType{Name: s.Name}
		if keyField != "" {
			entity.Key = &edmKey{PropertyRef: edmPropertyRef{Name: keyField}}
		}
		for _, f := range s.Fields {
			entity.Properties = append(entity.Properties, edmProperty{Name: f, Type: "Edm.String", Nullable: "true"})
		}
		schema.Entities = append(schema.Entities, entity)
		schema.Container.EntitySets = append(schema.Container.EntitySets, edmEntitySet{
			Name:       s.Name,
			EntityType: namespace + "." + s.Name,
		})
	}

	root := edmxRoot{Xmlns: "http://docs.oasis-open.org/odata/ns/edmx", Version: "4.0", Data: edmxData{Schema: schema}}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(root)
}

func (h *ODataHandler) serveEntitySet(w http.ResponseWriter, r *http.Request, entitySet, env, tenantID string) {
	sets, err := h.listEntitySets(r, env, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing entity sets: %v", err), http.StatusInternalServerError)
		return
	}
	var target *boEntitySet
	for i := range sets {
		if sets[i].Name == entitySet {
			target = &sets[i]
			break
		}
	}
	if target == nil {
		http.Error(w, fmt.Sprintf("entity set %q not found", entitySet), http.StatusNotFound)
		return
	}
	ep := target.Ep

	allowed, err := h.rateLimiter.Allow(r.Context(), tenantID, 1)
	if err != nil || !allowed {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	q := r.URL.Query()

	measures := target.Fields
	if sel := q.Get("$select"); sel != "" {
		fieldSet := make(map[string]bool, len(target.Fields))
		for _, f := range target.Fields {
			fieldSet[f] = true
		}
		var selected []string
		for _, f := range strings.Split(sel, ",") {
			f = strings.TrimSpace(f)
			if !fieldSet[f] {
				http.Error(w, fmt.Sprintf("$select: unknown field %q", f), http.StatusBadRequest)
				return
			}
			selected = append(selected, f)
		}
		measures = selected
	}

	filters, err := parseODataFilter(q.Get("$filter"))
	if err != nil {
		http.Error(w, fmt.Sprintf("$filter: %v", err), http.StatusBadRequest)
		return
	}

	top := 100
	if t := q.Get("$top"); t != "" {
		v, err := strconv.Atoi(t)
		if err != nil || v < 0 {
			http.Error(w, "$top must be a non-negative integer", http.StatusBadRequest)
			return
		}
		if v > 5000 {
			v = 5000
		}
		top = v
	}
	skip := 0
	if s := q.Get("$skip"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			http.Error(w, "$skip must be a non-negative integer", http.StatusBadRequest)
			return
		}
		skip = v
	}

	tenantUUID, _ := uuid.Parse(tenantID)
	reg := ""
	if rg, ok := region.GetRegionFromContext(r.Context()); ok {
		reg = rg
	}
	claims := jwtmiddleware.GetClaimsFromContext(r)

	req := analytics.BOSQLRequest{
		Env:                  env,
		TenantID:             &tenantUUID,
		BOName:               ep.BOName,
		EndpointID:           &ep.ID,
		Measures:             measures,
		Filters:              filters,
		CurrentUserID:        r.Header.Get("X-User-ID"),
		Region:               reg,
		CallerRoles:          claims.Roles,
		CallerOrganizationID: claims.OrganizationID,
	}

	planKey := GeneratePlanKey(ep.TenantID, ep.ID.String()+":odata", ep.Version, measures, filters, claims.Roles)
	plan, cacheErr := h.planCache.GetPlan(r.Context(), planKey)
	if cacheErr != nil || plan == nil {
		resolvedSQL, meta, err := h.resolver.ResolveQuery(r.Context(), req)
		if err != nil {
			if errors.Is(err, cbo.ErrEntitlementDenied) {
				http.Error(w, "Forbidden: caller does not satisfy entitlement policy", http.StatusForbidden)
				return
			}
			http.Error(w, fmt.Sprintf("Resolution error: %v", err), http.StatusInternalServerError)
			return
		}
		plan = &CachedPlan{SQL: resolvedSQL, MaskedFields: meta.MaskedFields}
		_ = h.planCache.SetPlan(r.Context(), planKey, *plan)
	}

	// $top/$skip aren't part of BOSQLRequest — wrap the resolved SQL as a
	// paged subquery. Safe to interpolate: top/skip are validated integers
	// above, never raw query text.
	pagedSQL := fmt.Sprintf("SELECT * FROM (%s) AS odata_page LIMIT %d OFFSET %d", plan.SQL, top, skip)

	var result []map[string]interface{}
	execErr := withTenantScopedQuery(r.Context(), h.db, tenantID, pagedSQL, func(rows *sqlx.Rows) error {
		for rows.Next() {
			row := make(map[string]interface{})
			if err := rows.MapScan(row); err != nil {
				return err
			}
			result = append(result, row)
		}
		return rows.Err()
	})
	if execErr != nil {
		http.Error(w, fmt.Sprintf("Execution error: %v", execErr), http.StatusInternalServerError)
		return
	}

	applyFieldMasking(result, plan.MaskedFields, claims.Roles)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"@odata.context": fmt.Sprintf("%s/$metadata#%s", h.mountPrefix, entitySet),
		"value":          result,
	})
}

// parseODataFilter supports only "field eq value" clauses ANDed together —
// e.g. "region eq 'US' and status eq 'active'". Anything else (gt/lt/
// contains/or/nested groups/functions) returns an error rather than being
// silently dropped, since a client that thinks it scoped a query and
// actually got everything is a real data exposure, not a UX nit.
func parseODataFilter(raw string) (map[string]interface{}, error) {
	filters := make(map[string]interface{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return filters, nil
	}

	clauses := strings.Split(raw, " and ")
	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		parts := strings.SplitN(clause, " eq ", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("unsupported filter clause %q — only \"field eq value\" clauses joined by \"and\" are supported", clause)
		}
		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
			value = strings.ReplaceAll(value[1:len(value)-1], "''", "'")
		}
		if field == "" {
			return nil, fmt.Errorf("unsupported filter clause %q", clause)
		}
		filters[field] = value
	}
	return filters, nil
}
