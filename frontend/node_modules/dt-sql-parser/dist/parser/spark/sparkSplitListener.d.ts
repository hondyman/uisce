import { SingleStatementContext } from '../../lib/spark/SparkSqlParser';
import { SparkSqlParserListener } from '../../lib/spark/SparkSqlParserListener';
import { SplitListener } from '../common/splitListener';
export declare class SparkSqlSplitListener extends SplitListener<SingleStatementContext> implements SparkSqlParserListener {
    exitSingleStatement(ctx: SingleStatementContext): void;
}
