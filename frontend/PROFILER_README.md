Profiler UI — usage and shortcuts

This document explains the profiling scopes and keyboard shortcuts in the frontend Profiler UI.

Overview
- The Profiler is available under the Semantic Mapper's "Profile" tab.
- It uses the tenant and datasource currently selected in the application (stored in localStorage keys `selected_tenant`, `selected_product`, `selected_datasource`).

Profiling scopes
- Table: Run the profiler for the single table currently selected in the Schema/Table selector.
- Schema: Run the profiler for all tables under the currently selected schema.
- Selected Tables: Select multiple tables using the checkboxes in the left Navigator and run the profiler for that chosen set.

How to run
1. Choose a schema from the Navigator or schema dropdown.
2. Select a table (or select multiple tables via checkboxes in the navigator).
3. Choose the desired Scope from the Scope dropdown (Table / Schema / Selected Tables).
4. Click "Run Profile" or press the 'p' key to start profiling for the selected scope.

Keyboard shortcuts
- 'p' : Run profile for the currently selected table (the new UI also supports 'selected' and 'schema' scopes; use the Run button to trigger these).

UI notes
- When profiling starts a small snackbar appears showing the chosen scope and how many nodes were scheduled.
- Profiled columns are highlighted in the columns table; results map back to the catalog node IDs via the `ColumnId` property in the profiler results.

Testing notes
- The integration tests mock fetch responses. In tests seed the tenant scope via localStorage using the same keys noted above.

If you want this README in a different location or with different formatting (e.g., included in the main docs), tell me where to place it.