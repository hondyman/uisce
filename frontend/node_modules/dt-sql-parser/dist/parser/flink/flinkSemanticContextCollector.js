import { FlinkSqlParser } from '../../lib/flink/FlinkSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class FlinkSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return FlinkSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return FlinkSqlParser.RULE_singleStatement;
    }
    enterSingleStatement(ctx) {
        this.visitStatement(ctx);
    }
}
export { FlinkSemanticContextCollector };
