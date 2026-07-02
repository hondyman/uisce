import { PostgreSqlParserListener } from '../../lib';
import { SingleStmtContext } from '../../lib/postgresql/PostgreSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class PostgreSemanticContextCollector extends SemanticContextCollector implements PostgreSqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterSingleStmt(ctx: SingleStmtContext): void;
}
export { PostgreSemanticContextCollector };
