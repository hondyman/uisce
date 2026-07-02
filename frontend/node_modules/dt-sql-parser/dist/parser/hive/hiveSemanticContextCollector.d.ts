import { HiveSqlParserListener } from '../../lib';
import { StatementContext } from '../../lib/hive/HiveSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class HiveSemanticContextCollector extends SemanticContextCollector implements HiveSqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterStatement(ctx: StatementContext): void;
}
export { HiveSemanticContextCollector };
