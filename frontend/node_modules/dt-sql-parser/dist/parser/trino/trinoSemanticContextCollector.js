import { TrinoSqlParser } from '../../lib/trino/TrinoSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class TrinoSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return TrinoSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return TrinoSqlParser.RULE_statements;
    }
    enterStatements(ctx) {
        this.visitStatement(ctx);
    }
}
export { TrinoSemanticContextCollector };
