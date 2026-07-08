---
description: A description of your rule
---

You are working on the Semantic OS. All code must adhere to the CARDINAL_RULES.md in the repository root. Before implementing any feature:

Check for Graph Anchors: Ensure you are not hardcoding logic that should be in the graph.

Validate UUIDs: All query parameters that translate to database UUIDs must be parsed with uuid.Parse. Never bind empty strings to UUID columns; handle blank inputs as NULL or skip the filter clause.

Inheritance Consistency: If querying catalog nodes or charge rates, implement the Gold Copy Inheritance Pattern (System Core + Tenant-Specific Custom Shadowing). Use ROW_NUMBER() OVER (PARTITION BY node_key ORDER BY precedence_rank DESC) to ensure custom records override baseline defaults.

Security First: Any lookup involving tenant_id or bo_id must use the secure multi-tenant union queries. Never perform a simple SELECT * WHERE tenant_id = ? without confirming the context is cryptographically anchored to the azp claim in the session.

Compile-Time Abstraction: SQL generators must resolve table names from the business_object_binding graph at compile time. Do not assume hardcoded table aliases."