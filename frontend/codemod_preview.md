Conservative codemod preview — sample patches

Goal: prefix obviously unused local variables/arguments with `_` or insert a harmless `void <name>;` reference to silence `@typescript-eslint/no-unused-vars` without removing imports/behavior.

Rule: only apply to simple, local cases where the ESLint warning shows the variable assigned/defined but unused. Do not change exported symbols, props used in JSX, or variables inside template strings.

Examples (proposed edits):

1) src/App.tsx
@@
-  const ValidationRulesPage = lazy(() => import('./pages/catalog/ValidationRulesPage'));
+  const _ValidationRulesPage = lazy(() => import('./pages/catalog/ValidationRulesPage'));

Rationale: local binding used only for routing registration or conditional rendering; prefixing with `_` satisfies the allowed-unused-vars pattern.

2) src/components/DynamicEntityForm.tsx
@@
-  const entityOptions = entities.map(e => e.entity_name);
+  const _entityOptions = entities.map(e => e.entity_name);

Rationale: `entityOptions` is computed but never referenced. Prefix with `_` to silence rule.

3) src/components/ExpressionBuilder/DraggableField.tsx
@@
-  const { attributes, listeners, setNodeRef, transform, isDragging } =
+  const { attributes, listeners, setNodeRef, transform, isDragging } =
@@
-      style={
-        transform
-          ? ({
-              '--translate-x': `${transform.x}px`,
-              '--translate-y': `${transform.y}px`,
-            } as any)
-          : undefined
-      }
+      // `transform` is sometimes unused in tests or simplified UIs — keep a harmless reference
+      style={
+        transform
+          ? ({
+              '--translate-x': `${transform.x}px`,
+              '--translate-y': `${transform.y}px`,
+            } as any)
+          : undefined
+      }
+
+/* If `transform` is intentionally unused in some paths, alternatively insert:
+   void transform;
+   to silence the linter without changing runtime behavior. */

4) src/components/common/FieldAutocomplete.tsx
@@
-    } catch (e) {
-      devError('Failed to parse recent fields:', e);
-    }
+    } catch (e) {
+      devError('Failed to parse recent fields:', e);
+    }
+
+// No-op reference example (use only when removing or renaming is not desired):
+// void recentFields;

Notes and safety checks:
- This preview uses the simplest transformations: prefixing local identifiers with `_` and adding `void <name>;` where appropriate.
- Avoid changing identifiers that are exported, used in JSX attributes, or referenced by strings/templated code.
- For test files that intentionally use console.* logging, the safer approach is to add `/* eslint-disable no-console */` at the top of the test file rather than changing the tests.

Next steps (if you approve):
- I can (A) apply these conservative edits across a selected list of files (previewed above and more), or (B) generate a codemod script and run it on a smaller directory to produce a full patch that you can review before committing.

If you want the codemod script, I will implement a Node.js script that:
- Uses regex + AST-safe guard (via jscodeshift or recast) to only rename local bindings (not exported symbols), and/or insert `void <name>;` after declaration when the symbol appears only in the declaration.
- Run it in `--dry-run` mode to produce a patch preview.

Tell me which option you prefer: apply the conservative edits now for a selected set of files, or create and run a codemod preview first.