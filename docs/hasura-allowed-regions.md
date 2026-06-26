# Hasura: Expose tenant.allowed_regions as session variable

Goal: ensure Hasura session contains `hasura.user.allowed_regions` (either CSV or JSON array), so GraphQL permissions and row filters can access tenant allowed regions and the UI can rely on `hasura.user.allowed_regions` for region-aware UI.

Recommended steps:

1. **Auth webhook / JWT claims**
   - If you use JWT-based auth, include an additional claim in the JWT payload under `https://hasura.io/jwt/claims` such as:

```json
"hasura": {
  "user_id": "<user id>",
  "x-hasura-tenant-id": "<tenant id>",
  "x-hasura-allowed-regions": "eu-west,us-east"
}
```

   - If you use an auth webhook (Hasura auth hook), have it add `hasura.user.allowed_regions` to the session it returns.

2. **Hasura permission templates**
   - Update table permissions / session usages to reference `current_setting('hasura.user.allowed_regions', true)` where needed in SQL views or permission checks.

3. **Optional: Hasura metadata changes**
   - If you want to surface `allowed_regions` on the `tenants` GraphQL object, add it as a database field (`allowed_regions JSONB`) (migration already added: `20260207_add_allowed_regions_to_tenants.up.sql`).
   - Add select permissions so the UI can query `tenants.allowed_regions` via GraphQL for tenant administration.

4. **Validation**
   - Ensure the auth layer populates `hasura.user.allowed_regions` as either a JSON array (preferred) or CSV string. The backend DB provider already accepts both formats.

Example (JWT claims):
```
{
  "https://hasura.io/jwt/claims": {
    "x-hasura-tenant-id": "910638ba-a459-4a3f-bb2d-78391b0595f6",
    "x-hasura-allowed-regions": "[\"eu-west\",\"us-east\"]"
  }
}
```

If you'd like, I can prepare a Hasura metadata patch (yaml/JSON) that adds `tenants.allowed_regions` to the GraphQL schema and an example permission template that checks `hasura.user.allowed_regions`.
