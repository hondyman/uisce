import { MySqlParserListener } from '../../lib';
import { SingleStatementContext } from '../../lib/mysql/MySqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class MySqlSemanticContextCollector extends SemanticContextCollector implements MySqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterSingleStatement(ctx: SingleStatementContext): void;
}
export { MySqlSemanticContextCollector };
