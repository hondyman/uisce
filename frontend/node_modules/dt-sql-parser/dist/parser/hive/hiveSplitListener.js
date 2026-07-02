import { SplitListener } from '../common/splitListener';
export class HiveSqlSplitListener extends SplitListener {
    exitStatement(ctx) {
        this._statementsContext.push(ctx);
    }
}
