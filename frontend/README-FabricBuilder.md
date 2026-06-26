# Fabric Builder

A unified semantic model editor with canvas, code view, and Extends management.

## Highlights
- Canvas tiles for dimensions, measures, filters, joins, and an Extends tile for custom models.
- Typeahead Extends form with accessibility-friendly combobox semantics.
- Code/Canvas toggle with JSON/YAML formats and quick-jump to sections.
- Folder-aware Views Catalog under the Fabric menu (CRUD and preview).

## Extends behavior
- The Extends tile appears for custom models that extend a base model.
- Clicking the Extends tile opens the Extends form in the right panel.
- By default, Extends can be changed outside of edit mode for smoother UX.
  - To require edit mode, pass `allowExtendsChangeOutsideEdit={false}` to `SemanticModelEditor`.
- The form prevents invalid selections (self-extend and idempotent changes).

## Accessibility
- Extends form is a proper ARIA combobox:
  - role="combobox", aria-autocomplete="list", aria-expanded, aria-controls, and aria-activedescendant.
  - Keyboard support: ArrowUp/Down, Enter, Escape.

## Testing
- Unit tests cover canvas interactions, Extends change flows, conflict flows, and accessibility.
- Run tests:

```
npm test -- --reporter=verbose
```

## Build

```
npm run build
```

Artifacts are output to `frontend/dist/`.
