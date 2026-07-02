import { StatementContext } from '../../lib/hive/HiveSqlParser';
import { HiveSqlParserListener } from '../../lib/hive/HiveSqlParserListener';
import { SplitListener } from '../common/splitListener';
export declare class HiveSqlSplitListener extends SplitListener<StatementContext> implements HiveSqlParserListener {
    exitStatement(ctx: StatementContext): void;
}
