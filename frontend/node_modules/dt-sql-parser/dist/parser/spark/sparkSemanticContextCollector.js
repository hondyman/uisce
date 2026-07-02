import { SparkSqlParser } from '../../lib/spark/SparkSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class SparkSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return SparkSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return SparkSqlParser.RULE_singleStatement;
    }
    enterSingleStatement(ctx) {
        this.visitStatement(ctx);
    }
}
export { SparkSemanticContextCollector };
