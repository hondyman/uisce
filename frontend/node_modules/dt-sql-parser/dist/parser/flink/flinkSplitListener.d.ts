import { SingleStatementContext } from '../../lib/flink/FlinkSqlParser';
import { FlinkSqlParserListener } from '../../lib/flink/FlinkSqlParserListener';
import { SplitListener } from '../common/splitListener';
export declare class FlinkSqlSplitListener extends SplitListener<SingleStatementContext> implements FlinkSqlParserListener {
    exitSingleStatement(ctx: SingleStatementContext): void;
}
