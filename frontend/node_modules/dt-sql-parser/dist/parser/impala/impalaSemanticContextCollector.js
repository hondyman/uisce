import { ImpalaSqlParser } from '../../lib/impala/ImpalaSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
class ImpalaSemanticContextCollector extends SemanticContextCollector {
    getWhiteSpaceRuleType() {
        return ImpalaSqlParser.WHITE_SPACE;
    }
    getStatementRuleType() {
        return ImpalaSqlParser.RULE_singleStatement;
    }
    enterSingleStatement(ctx) {
        this.visitStatement(ctx);
    }
}
export { ImpalaSemanticContextCollector };
