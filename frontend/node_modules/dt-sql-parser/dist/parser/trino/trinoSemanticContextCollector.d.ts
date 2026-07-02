import { TrinoSqlListener } from '../../lib';
import { StatementsContext } from '../../lib/trino/TrinoSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class TrinoSemanticContextCollector extends SemanticContextCollector implements TrinoSqlListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterStatements(ctx: StatementsContext): void;
}
export { TrinoSemanticContextCollector };
