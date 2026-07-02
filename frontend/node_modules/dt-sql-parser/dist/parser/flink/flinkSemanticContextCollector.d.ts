import { FlinkSqlParserListener } from '../../lib';
import { SingleStatementContext } from '../../lib/flink/FlinkSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class FlinkSemanticContextCollector extends SemanticContextCollector implements FlinkSqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterSingleStatement(ctx: SingleStatementContext): void;
}
export { FlinkSemanticContextCollector };
