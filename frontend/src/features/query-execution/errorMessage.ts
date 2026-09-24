/**
 * Shared error normalization for the query-execution layer.
 *
 * The multi-BO join resolver (boresolver / multi_bo_generator.go) speaks a
 * different relationship graph than the BO CRUD `/relationships` endpoint
 * the editor's "+ Join" uses to populate related-object candidates - a
 * related table can be legitimately joinable for a Page Studio widget but
 * still have no resolvable path for the query engine's SQL generator. That
 * mismatch surfaces as a raw Go error ("no relationship path from X to Y:
 * failed to resolve driving table..."); this turns it into something a user
 * can actually act on instead of a stack-trace-shaped string.
 *
 * Lifted verbatim from SavedQueryEditor so every consumer of the
 * useQueryExecution hook renders identical error text.
 */
export function friendlyQueryError(message: string): string {
  if (/no relationship path|failed to resolve driving table/i.test(message)) {
    return 'One of the joined related objects can\'t be resolved into SQL by the query engine ' +
      '(it may only support drag-and-drop placement, not filtering/grouping). Remove it from the ' +
      'query and try again, or ask for it to be added to the query engine\'s join graph.';
  }
  return message;
}
