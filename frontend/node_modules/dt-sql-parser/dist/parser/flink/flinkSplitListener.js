import { SplitListener } from '../common/splitListener';
export class FlinkSqlSplitListener extends SplitListener {
    exitSingleStatement(ctx) {
        this._statementsContext.push(ctx);
    }
}
