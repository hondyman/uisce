import { SplitListener } from '../common/splitListener';
export class TrinoSqlSplitListener extends SplitListener {
    exitSingleStatement(ctx) {
        this._statementsContext.push(ctx);
    }
}
