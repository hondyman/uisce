package proxy

import "github.com/gin-gonic/gin"

type RouteRegistrar struct {
	handler *ProxyHandler
	api     *gin.RouterGroup
}

func NewRouteRegistrar(handler *ProxyHandler, api *gin.RouterGroup) *RouteRegistrar {
	return &RouteRegistrar{
		handler: handler,
		api:     api,
	}
}

func (r *RouteRegistrar) RegisterAll() {
	r.registerSemanticRoutes()
	r.registerBusinessTermRoutes()
	r.registerInstanceRoutes()
	r.registerImpactRoutes()
	r.registerBundleRoutes()
	r.registerModelRoutes()
	r.registerAuthRoutes()
	r.registerFabricRoutes()
	r.registerTenantRoutes()
	r.registerIPWhitelistRoutes()
	r.registerQueryRoutes()
	r.registerViewRoutes()
	r.registerCalculationRoutes()
	r.registerRoleRoutes()
	r.registerPolicyRoutes()
	r.registerProfilerRoutes()
	r.registerDataDomainRoutes()
	r.registerEntitySchemaRoutes()
	r.registerValidationRuleRoutes()
	r.registerRelationshipRoutes()
	r.registerCatalogRoutes()
	r.registerLineageRoutes()
	r.registerNodeTypeRoutes()
	r.registerEdgeTypeRoutes()
	r.registerBPNotificationRoutes()
}

func (r *RouteRegistrar) registerSemanticRoutes() {
	r.api.GET("/semantic/objects", r.handler.ServeHTTP())
	r.api.Any("/semantic-mapping", r.handler.ServeHTTP())
	r.api.Any("/semantic-mapping/*path", r.handler.ServeHTTP())
	r.api.Any("/semantic-mappings", r.handler.ServeHTTP())
	r.api.Any("/semantic-mappings/*path", r.handler.ServeHTTP())
	r.api.Any("/semantic-terms", r.handler.ServeHTTP())
	r.api.Any("/semantic-terms/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerBusinessTermRoutes() {
	r.api.Any("/business-term", r.handler.ServeHTTP())
	r.api.Any("/business-terms", r.handler.ServeHTTP())
	r.api.Any("/business-terms/*path", r.handler.ServeHTTP())
	r.api.Any("/business-term-edges", r.handler.ServeHTTP())
	r.api.Any("/business-term-edges/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerInstanceRoutes() {
	r.api.Any("/instance", r.handler.ServeHTTP())
	r.api.Any("/instance/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerImpactRoutes() {
	r.api.Any("/impact", r.handler.ServeHTTP())
	r.api.Any("/impact/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerBundleRoutes() {
	r.api.GET("/bundles", r.handler.ServeHTTP())
	r.api.POST("/bundles", r.handler.ServeHTTP())
	r.api.GET("/bundles/:id", r.handler.ServeHTTP())
	r.api.PUT("/bundles/:id", r.handler.ServeHTTP())
	r.api.Any("/bundles/:id/*any", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerModelRoutes() {
	r.api.Any("/models", r.handler.ServeHTTP())
	r.api.POST("/models/generated", r.handler.ServeHTTP())
	r.api.POST("/models/custom", r.handler.ServeHTTP())
	r.api.POST("/models/clone", r.handler.ServeHTTP())
	r.api.Any("/models/:model_id", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerAuthRoutes() {
	r.api.POST("/auth/logout", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerFabricRoutes() {
	r.api.Any("/fabric/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerTenantRoutes() {
	r.api.Any("/tenants", r.handler.ServeHTTP())
	r.api.Any("/tenants/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerIPWhitelistRoutes() {
	r.api.Any("/ip-whitelist", r.handler.ServeHTTP())
	r.api.Any("/ip-whitelist/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerQueryRoutes() {
	r.api.Any("/query", r.handler.ServeHTTP())
	r.api.Any("/pre_aggregations", r.handler.ServeHTTP())
	r.api.Any("/pre_aggregations/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerViewRoutes() {
	r.api.Any("/views", r.handler.ServeHTTP())
	r.api.Any("/views/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerCalculationRoutes() {
	r.api.Any("/calculations", r.handler.ServeHTTP())
	r.api.Any("/calculations/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerRoleRoutes() {
	r.api.Any("/roles", r.handler.ServeHTTP())
	r.api.Any("/roles/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerPolicyRoutes() {
	r.api.Any("/policies", r.handler.ServeHTTP())
	r.api.Any("/policies/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerProfilerRoutes() {
	r.api.Any("/profiler", r.handler.ServeHTTP())
	r.api.Any("/profiler/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerDataDomainRoutes() {
	r.api.Any("/data-domains", r.handler.ServeHTTP())
	r.api.Any("/data-domains/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerEntitySchemaRoutes() {
	r.api.GET("/entity-schema", r.handler.ServeHTTP())
	r.api.POST("/entity-schema", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerValidationRuleRoutes() {
	r.api.GET("/validation-rules", r.handler.ServeHTTP())
	r.api.POST("/validation-rules", r.handler.ServeHTTP())
	r.api.GET("/validation-rules/:id", r.handler.ServeHTTP())
	r.api.PATCH("/validation-rules/:id", r.handler.ServeHTTP())
	r.api.DELETE("/validation-rules/:id", r.handler.ServeHTTP())
	r.api.POST("/validation-rules/:id/execute", r.handler.ServeHTTP())
	r.api.POST("/validation-rules/execute-batch", r.handler.ServeHTTP())
	r.api.GET("/validation-rules/:id/audit", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerRelationshipRoutes() {
	r.api.GET("/schema/:entity", r.handler.ServeHTTP())
	r.api.POST("/rules/test", r.handler.ServeHTTP())
	r.api.GET("/ai/discover-relationships/:entityId", r.handler.ServeHTTP())
	r.api.POST("/ai/generate-rule", r.handler.ServeHTTP())
	r.api.Any("/relationships", r.handler.ServeHTTP())
	r.api.Any("/relationships/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerCatalogRoutes() {
	r.api.Any("/catalog/tables", r.handler.ServeHTTP())
	r.api.Any("/catalog/tables/*path", r.handler.ServeHTTP())
	r.api.Any("/catalog/nodes", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerLineageRoutes() {
	r.api.Any("/lineage", r.handler.ServeHTTP())
	r.api.Any("/lineage/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerNodeTypeRoutes() {
	r.api.Any("/node-types", r.handler.ServeHTTP())
	r.api.Any("/node-types/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerEdgeTypeRoutes() {
	r.api.Any("/edge-types", r.handler.ServeHTTP())
	r.api.Any("/edge-types/*path", r.handler.ServeHTTP())
}

func (r *RouteRegistrar) registerBPNotificationRoutes() {
	r.api.Any("/bp-notifications", r.handler.ServeHTTP())
	r.api.Any("/bp-notifications/*path", r.handler.ServeHTTP())
}
