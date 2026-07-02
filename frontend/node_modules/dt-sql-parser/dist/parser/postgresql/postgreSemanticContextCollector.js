import { PostgreSqlParser } from '../../lib/postgresql/PostgreSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class PostgreSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return PostgreSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return PostgreSqlParser.RULE_singleStmt;
    }
    enterSingleStmt(ctx) {
        this.visitStatement(ctx);
    }
}
export { PostgreSemanticContextCollector };
