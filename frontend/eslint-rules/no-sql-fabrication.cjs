/**
 * no-sql-fabrication - structural detection of client-side SQL synthesis.
 *
 * Layer 2 of the two-layer guardrail (see eslint.config.cjs). Layer 1 bans
 * by naming convention; this rule catches interpolated SQL-shaped strings
 * even when they aren't named like a generator. The discriminating feature
 * is structural: UI copy is static; SQL fabrication has interpolation
 * holes or concatenation with dynamic operands.
 *
 * Triggers:
 *   - TemplateLiteral with >= 2 interpolation holes, raw text matches a
 *     SQL shape. Filters out static UI copy ("Select all from this list")
 *     and short single-hole strings ("Pick ${first} from the team").
 *   - BinaryExpression `+` chain whose flattened operands include at least
 *     one SQL-shaped literal AND at least one non-literal dynamic operand.
 *     Catches `'SELECT ' + cols + ' FROM ' + table`.
 *
 * Severity: warn everywhere, error nowhere - the work list is the
 * FilterBuilderPanel Phase 3 migration, and warnings are the migration's
 * surface area.
 */
const SQL_SHAPE = [
  /\bSELECT\b[\s\S]{0,600}?\bFROM\b/i,
  /\bINSERT\s+INTO\b/i,
  /\bDELETE\s+FROM\b/i,
  /\bUPDATE\b\s+\S+\s+\bSET\b/i,
  /\b(LEFT|RIGHT|INNER|FULL)(\s+OUTER)?\s+JOIN\b/i,
  /\bGROUP\s+BY\b/i,
  /\b(HAVING|QUALIFY)\b/i,
];

function isSqlShaped(s) {
  return typeof s === 'string' && s.length >= 12 && SQL_SHAPE.some((re) => re.test(s));
}

function flattenConcat(node, out) {
  if (node.type === 'BinaryExpression' && node.operator === '+') {
    flattenConcat(node.left, out);
    flattenConcat(node.right, out);
  } else {
    out.push(node);
  }
}

module.exports = {
  meta: {
    type: 'problem',
    schema: [],
    messages: {
      fabrication:
        'Possible client-side SQL fabrication (interpolated SQL-shaped string). SQL must come ' +
        'from the backend (previewQuery/executeQuery). If this is UI copy, reword it. If this ' +
        'is FilterBuilderPanel migration work, this warning is your Phase 3 work list.',
    },
  },
  create(context) {
    const report = (node) => context.report({ node, messageId: 'fabrication' });

    return {
      TemplateLiteral(node) {
        // Require >= 2 holes: UI copy is static; real fabrication is built
        // up. One-hole strings are usually things like "Pick ${first} from
        // the team" - not SQL.
        if (node.expressions.length < 2) return;
        const raw = node.quasis.map((q) => q.value.raw).join('?');
        if (isSqlShaped(raw)) report(node);
      },
      BinaryExpression(node) {
        if (node.operator !== '+') return;
        const parts = [];
        flattenConcat(node, parts);
        const hasSqlLiteral = parts.some(
          (p) => p.type === 'Literal' && isSqlShaped(p.value)
        );
        const hasDynamic = parts.some(
          (p) => p.type !== 'Literal'
        );
        if (hasSqlLiteral && hasDynamic) report(node);
      },
    };
  },
};
