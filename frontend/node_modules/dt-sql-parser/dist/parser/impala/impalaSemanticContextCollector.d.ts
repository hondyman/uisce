import { ImpalaSqlParserListener } from '../../lib';
import { SingleStatementContext } from '../../lib/impala/ImpalaSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class ImpalaSemanticContextCollector extends SemanticContextCollector implements ImpalaSqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterSingleStatement(ctx: SingleStatementContext): void;
}
export { ImpalaSemanticContextCollector };
