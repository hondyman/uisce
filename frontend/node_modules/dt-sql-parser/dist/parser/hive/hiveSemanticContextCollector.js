import { HiveSqlParser } from '../../lib/hive/HiveSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class HiveSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return HiveSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return HiveSqlParser.RULE_statement;
    }
    enterStatement(ctx) {
        this.visitStatement(ctx);
    }
}
export { HiveSemanticContextCollector };
