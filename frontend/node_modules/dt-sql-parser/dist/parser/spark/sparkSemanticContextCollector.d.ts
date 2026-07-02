import { SparkSqlParserListener } from '../../lib';
import { SingleStatementContext } from '../../lib/spark/SparkSqlParser';
import SemanticContextCollector from '../common/semanticContextCollector';
declare class SparkSemanticContextCollector extends SemanticContextCollector implements SparkSqlParserListener {
    getWhiteSpaceRuleType(): number;
    getStatementRuleType(): number;
    enterSingleStatement(ctx: SingleStatementContext): void;
}
export { SparkSemanticContextCollector };
