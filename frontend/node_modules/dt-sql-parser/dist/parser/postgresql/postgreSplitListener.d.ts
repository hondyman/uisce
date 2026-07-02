import { SingleStmtContext } from '../../lib/postgresql/PostgreSqlParser';
import { PostgreSqlParserListener } from '../../lib/postgresql/PostgreSqlParserListener';
import { SplitListener } from '../common/splitListener';
export declare class PostgreSqlSplitListener extends SplitListener<SingleStmtContext> implements PostgreSqlParserListener {
    exitSingleStmt(ctx: SingleStmtContext): void;
}
