import { SplitListener } from '../common/splitListener';
export class PostgreSqlSplitListener extends SplitListener {
    exitSingleStmt(ctx) {
        this._statementsContext.push(ctx);
    }
}
