# Phase 1: Core Calendar Management - Implementation Checklist

**Target Timeline**: 1 week (5 business days)

**Team**: 2-3 developers (1 backend, 1 frontend, 1 optional QA/devops)

---

## Day 1: Database & Hasura Setup

### Morning (2-3 hours)

- [ ] **1.1** Connect to remote PostgreSQL
  - [ ] Test connection with `psql`
  - [ ] Confirm admin credentials work
  - [ ] Document connection string

- [ ] **1.2** Run schema migration
  - [ ] Copy `docs/schema.sql` to remote server
  - [ ] Execute: `psql < schema.sql`
  - [ ] Verify tables created: `\dt`
  - [ ] Verify indexes created: `\di`

- [ ] **1.3** Verify RLS enabled
  - [ ] Check: `SELECT tablename FROM pg_tables WHERE rowsecurity;`
  - [ ] Expected: calendars, schedule_profiles, blackouts, audit_log all have RLS

- [ ] **1.4** Insert test data (optional)
  - [ ] Copy-paste example data from schema.sql
  - [ ] Verify: `SELECT * FROM calendars LIMIT 1;`

### Afternoon (3-4 hours)

- [ ] **1.5** Connect Hasura to PostgreSQL
  - [ ] Open Hasura Console
  - [ ] Add PostgreSQL database connection
  - [ ] Test connection
  - [ ] Document endpoint and credentials

- [ ] **1.6** Track tables in Hasura
  - [ ] Go to **Data** → **Databases** → track each table:
    - [ ] calendars
    - [ ] schedule_profiles
    - [ ] profile_calendars
    - [ ] blackouts
    - [ ] audit_log
  - [ ] Verify tables appear in GraphiQL

- [ ] **1.7** Configure RLS in Hasura
  - [ ] For `calendars` table:
    - [ ] Go to **Permissions** tab
    - [ ] Select "User" role (or create)
    - [ ] Add filter: `tenant_id` equals `X-Hasura-Tenant-Id`
    - [ ] Save permissions
  - [ ] Repeat for: schedule_profiles, blackouts, audit_log

- [ ] **1.8** Test GraphQL query in Hasura Console
  - [ ] Run sample query
  - [ ] Add header: `X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000`
  - [ ] Expected: Returns only data for that tenant

**Day 1 Success Criteria**:
- ✅ Schema created on remote PostgreSQL
- ✅ RLS enabled on all tables
- ✅ Hasura connected
- ✅ Tables tracked
- ✅ RLS permissions configured
- ✅ Sample GraphQL query works

---

## Day 2: Golang Service Implementation (Part 1 - Setup & GET endpoints)

### Morning (3-4 hours)

- [ ] **2.1** Update local development environment
  - [ ] Edit `.env.local` with real remote credentials:
    - [ ] HASURA_ENDPOINT
    - [ ] HASURA_ADMIN_SECRET
    - [ ] POSTGRES_* credentials
  - [ ] Verify each variable is set: `cat .env.local`

- [ ] **2.2** Start local services
  - [ ] Run: `make dev`
  - [ ] Expected output:
    - [ ] Redpanda started
    - [ ] Calendar Service on port 8081
    - [ ] Connected to remote Hasura

- [ ] **2.3** Test health check
  - [ ] Run: `curl http://localhost:8081/health`
  - [ ] Expected: `{"status":"healthy"}`

- [ ] **2.4** Implement `ListCalendars` in `calendar_service.go`
  - [ ] Write GraphQL query in service
  - [ ] Parse results
  - [ ] Return []Calendar
  - [ ] Handle errors
  - [ ] Add logging

- [ ] **2.5** Implement `GetCalendarByID` in `calendar_service.go`
  - [ ] Query by ID + tenant_id
  - [ ] Verify bitemporal filtering (valid_to IS NULL)
  - [ ] Return single Calendar

### Afternoon (3-4 hours)

- [ ] **2.6** Implement GET handlers in `api/calendar_handlers.go`
  - [ ] `GET /api/v1/calendars` - list with pagination
    - [ ] Parse query params (limit, offset, region)
    - [ ] Validate limit (max 1000)
    - [ ] Call service
    - [ ] Return JSON response with pagination info
  
  - [ ] `GET /api/v1/calendars/:id` - fetch single
    - [ ] Parse ID from URL
    - [ ] Extract tenant from header
    - [ ] Call service
    - [ ] Handle 404 error
    - [ ] Return JSON response

- [ ] **2.7** Test GET endpoints with curl
  - [ ] `curl http://localhost:8081/api/v1/calendars` (should be empty initially)
  - [ ] Add sample calendar to DB manually (via psql)
  - [ ] Test again (should return 1 calendar)
  - [ ] Test pagination: `?limit=5&offset=0`
  - [ ] Test by ID: `/api/v1/calendars/{id}`

- [ ] **2.8** Add logging & metrics
  - [ ] Log each request (tenant_id, request_id, duration)
  - [ ] Add Prometheus counter: calendar_list_total, calendar_get_total
  - [ ] Test: `curl http://localhost:8081/metrics`

**Day 2 Success Criteria**:
- ✅ Local service connected to remote Hasura
- ✅ GET /api/v1/calendars working
- ✅ GET /api/v1/calendars/:id working
- ✅ Pagination working
- ✅ Curl tests passing
- ✅ Logging working

---

## Day 3: Golang Service Implementation (Part 2 - CREATE & UPDATE)

### Morning (3-4 hours)

- [ ] **3.1** Implement `CreateCalendar` in `calendar_service.go`
  - [ ] Validate inputs (name, region, holidays JSON)
  - [ ] Generate UUID for new calendar
  - [ ] Execute Hasura mutation (insert_calendars_one)
  - [ ] Create audit entry (call audit service)
  - [ ] Publish event to Redpanda (optional, for Phase 3)
  - [ ] Return created calendar

- [ ] **3.2** Implement `UpdateCalendar` (bitemporal) in `calendar_service.go`
  - [ ] **Key**: This is BITEMPORAL - don't update existing rows
  - [ ] Fetch current active version (valid_to IS NULL)
  - [ ] Execute two-step mutation:
    - Step 1: Set valid_to = NOW() on old version
    - Step 2: Insert new version with updated fields
  - [ ] Return new version
  - [ ] Create audit entry with old_values + new_values

- [ ] **3.3** Implement `DeleteCalendar` (soft delete) in `calendar_service.go`
  - [ ] Set valid_to = NOW()
  - [ ] Create audit entry
  - [ ] Return success (no error)

### Afternoon (3-4 hours)

- [ ] **3.4** Implement POST handler in `api/calendar_handlers.go`
  - [ ] `POST /api/v1/calendars` - create
    - [ ] Parse JSON body
    - [ ] Validate required fields
    - [ ] Validate holiday JSON format
    - [ ] Extract tenant from header
    - [ ] Call service.CreateCalendar()
    - [ ] Return 201 Created with location header

- [ ] **3.5** Implement PATCH handler in `api/calendar_handlers.go`
  - [ ] `PATCH /api/v1/calendars/:id` - update
    - [ ] Parse ID from URL
    - [ ] Parse JSON body (all fields optional)
    - [ ] Extract tenant from header
    - [ ] Verify calendar belongs to tenant
    - [ ] Call service.UpdateCalendar()
    - [ ] Return 200 with new version

- [ ] **3.6** Implement DELETE handler in `api/calendar_handlers.go`
  - [ ] `DELETE /api/v1/calendars/:id` - delete
    - [ ] Parse ID
    - [ ] Extract tenant
    - [ ] Call service.DeleteCalendar()
    - [ ] Return 204 No Content

- [ ] **3.7** Test CRUD operations with curl
  ```bash
  # Create
  curl -X POST http://localhost:8081/api/v1/calendars \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test","region":"US","holidays":[]}'
  
  # Verify created (list)
  curl http://localhost:8081/api/v1/calendars \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
  
  # Update
  curl -X PATCH http://localhost:8081/api/v1/calendars/{ID} \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000" \
    -H "Content-Type: application/json" \
    -d '{"name":"Updated"}'
  
  # Verify update changed name (get)
  curl http://localhost:8081/api/v1/calendars/{ID} \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
  
  # Delete
  curl -X DELETE http://localhost:8081/api/v1/calendars/{ID} \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
  
  # Verify deleted (should return empty list)
  curl http://localhost:8081/api/v1/calendars \
    -H "X-Hasura-Tenant-Id: 550e8400-e29b-41d4-a716-446655440000"
  ```

**Day 3 Success Criteria**:
- ✅ POST /api/v1/calendars working (201 Created)
- ✅ PATCH /api/v1/calendars/:id working (bitemporal versioning)
- ✅ DELETE /api/v1/calendars/:id working (soft delete)
- ✅ All curl tests passing
- ✅ Audit log entries created for all operations
- ✅ Bitemporal versioning verified (old version has valid_to, new has valid_from)

---

## Day 4: React Frontend (Part 1 - Setup & Components)

### Morning (3-4 hours)

- [ ] **4.1** Ensure React project is set up
  - [ ] Navigate to frontend directory
  - [ ] Run `npm install` (if not done)
  - [ ] Verify build: `npm run build`

- [ ] **4.2** Create GraphQL queries/mutations
  - [ ] Create `src/graphql/calendars.ts`:
    - [ ] Query: LIST_CALENDARS
    - [ ] Query: GET_CALENDAR
    - [ ] Mutation: CREATE_CALENDAR
    - [ ] Mutation: UPDATE_CALENDAR
    - [ ] Mutation: DELETE_CALENDAR

- [ ] **4.3** Generate Apollo hooks (optional, if using graphql-codegen)
  - [ ] Run: `npm run graphql:generate`
  - [ ] Verify hooks generated

### Afternoon (3-4 hours)

- [ ] **4.4** Create `CalendarList` component
  - [ ] Use Apollo useQuery for LIST_CALENDARS
  - [ ] Display as table (Ant Design Table)
  - [ ] Show: ID, Name, Region, Valid From, Actions
  - [ ] Add loading, error states
  - [ ] Add pagination controls

- [ ] **4.5** Create `CalendarForm` component
  - [ ] Form fields: name, region, holidays (JSON editor)
  - [ ] Validation:
    - [ ] name required, max 255
    - [ ] holidays valid JSON array
  - [ ] Submit → call mutation (CREATE or UPDATE)
  - [ ] Show success/error messages

- [ ] **4.6** Create `Calendars` page
  - [ ] Layout: Top bar with "Create Calendar" button
  - [ ] Main area: CalendarList
  - [ ] Modal: CalendarForm (for create/edit)
  - [ ] Add refresh button

- [ ] **4.7** Implement CRUD interactions
  - [ ] Click "Create" → open form modal
  - [ ] Submit form → call CREATE_CALENDAR mutation
  - [ ] On success → refresh list, close modal
  - [ ] Click edit → populate form, call UPDATE
  - [ ] Click delete → confirmation, call DELETE

**Day 4 Success Criteria**:
- ✅ CalendarList component rendering
- ✅ GraphQL queries working (connected to Golang backend)
- ✅ Table showing calendars from database
- ✅ Pagination working
- ✅ Create button opens form
- ✅ Edit/delete buttons work

---

## Day 5: Testing, Documentation, & Integration

### Morning (3-4 hours)

- [ ] **5.1** Write unit tests (Golang)
  - [ ] Test CreateCalendar() with valid/invalid inputs
  - [ ] Test UpdateCalendar() bitemporal logic
  - [ ] Test DeleteCalendar() soft delete
  - [ ] Test ListCalendars() pagination
  - [ ] Run: `go test -v ./...`
  - [ ] Target: >80% code coverage

- [ ] **5.2** Write integration tests (Golang)
  - [ ] Test full flow: create → read → update → list → delete
  - [ ] Test tenant isolation (calendar from tenant A not visible to tenant B)
  - [ ] Test error cases (invalid input, not found, etc.)
  - [ ] Run against local services

- [ ] **5.3** Write React component tests
  - [ ] Test CalendarList renders
  - [ ] Test form validation
  - [ ] Test CRUD button interactions
  - [ ] Run: `npm test`

### Afternoon (2-3 hours)

- [ ] **5.4** Create Postman collection
  - [ ] Document all 5 endpoints (GET, GET/:id, POST, PATCH, DELETE)
  - [ ] Include examples
  - [ ] Export as JSON

- [ ] **5.5** Update documentation
  - [ ] Add implementation notes to QUICKSTART.md
  - [ ] Update API.md with actual examples
  - [ ] Add known issues/TODOs

- [ ] **5.6** End-to-end verification
  - [ ] Start all services: `make dev`
  - [ ] Open React app: http://localhost:3000
  - [ ] Create a calendar via UI
  - [ ] Verify appears in list
  - [ ] Edit it (update name)
  - [ ] Verify changed
  - [ ] Delete it
  - [ ] Verify removed
  - [ ] Check audit log in DB

- [ ] **5.7** Performance baseline
  - [ ] Measure: List 1000 calendars
  - [ ] Measure: Create calendar (latency)
  - [ ] Document baseline metrics

**Day 5 Success Criteria**:
- ✅ All unit tests passing (>80% coverage)
- ✅ All integration tests passing
- ✅ React components tests passing
- ✅ E2E workflow verified
- ✅ Documentation complete
- ✅ Postman collection created
- ✅ Performance baseline established

---

## Acceptance Criteria (Phase 1 Definition of Done)

- [ ] All CRUD operations working (backend + frontend)
- [ ] Tests passing (unit, integration, component)
- [ ] Bitemporal versioning verified
- [ ] Audit log working for all operations
- [ ] RLS/tenant isolation enforced
- [ ] Error handling comprehensive
- [ ] Documentation complete (API, deployment, troubleshooting)
- [ ] Production Docker image builds
- [ ] Local Docker Compose environment stable
- [ ] Performance acceptable (<500ms for list, <200ms for single ops)
- [ ] Security checklist complete (no secrets in code, HTTPS ready, etc.)

---

## Blockers & Escalations

If you encounter any of these, document and escalate:

- [ ] Cannot connect to remote PostgreSQL (credentials, firewall)
- [ ] Hasura GraphQL queries returning errors (RLS misconfiguration)
- [ ] Debezium not running (Docker network issues)
- [ ] Apollo client not connecting to Golang backend (CORS issues)
- [ ] Performance issues (query optimization needed)
- [ ] Schema conflicts with existing data (migration strategy)

---

## Success Metrics

By end of Day 5:

| Metric | Target | Actual |
|--------|--------|--------|
| Endpoints implemented | 5/5 | |
| Test coverage | >80% | |
| Unit tests passing | 100% | |
| Integration tests passing | 100% | |
| React components rendering | 3/3 | |
| E2E workflow verified | ✅ | |
| Documentation pages | 4 | |
| Production image builds | ✅ | |
| Deployment ready | ✅ | |

---

## Next Phase Preparation

Before moving to Phase 2 (Availability Checking):

- [ ] Phase 1 code reviewed and merged
- [ ] Performance baseline documented
- [ ] Known issues logged as GitHub Issues
- [ ] Team trained on bitemporal versioning pattern
- [ ] Redpanda CDC pipeline verified (prep for Phase 3)

---

## Notes & Tips

1. **Bitemporal Versioning**: Don't update rows directly. Always INSERT new versions with closed old versions.
2. **Tenant Isolation**: Always validate tenant_id from header matches records.
3. **Pagination**: Use LIMIT + OFFSET, not cursor-based (simpler for Phase 1).
4. **Logging**: Log every request with request_id for tracing.
5. **Testing**: Test with multiple tenants to verify isolation.
6. **Git**: Commit after each day's work with clear messages.

Good luck! 🚀
