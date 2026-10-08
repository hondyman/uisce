package provisioning

// ValidTenantCode reports whether code is an acceptable tenant code. It applies
// the same rule the provisioning handler enforces, so code outside the handler
// (identity, secrets) validates a tenant code the same way.
func ValidTenantCode(code string) bool {
	return codePattern.MatchString(code)
}
