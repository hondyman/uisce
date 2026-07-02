import { MySqlParser } from '../../lib/mysql/MySqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class MySqlSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return MySqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return MySqlParser.RULE_singleStatement;
    }
    enterSingleStatement(ctx) {
        this.visitStatement(ctx);
    }
}
export { MySqlSemanticContextCollector };
