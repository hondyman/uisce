import { SingleStatementContext } from '../../lib/trino/TrinoSqlParser';
import { TrinoSqlListener } from '../../lib/trino/TrinoSqlListener';
import { SplitListener } from '../common/splitListener';
export declare class TrinoSqlSplitListener extends SplitListener<SingleStatementContext> implements TrinoSqlListener {
    exitSingleStatement(ctx: SingleStatementContext): void;
}
