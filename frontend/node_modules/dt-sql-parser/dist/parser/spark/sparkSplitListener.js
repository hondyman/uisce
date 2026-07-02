import { SplitListener } from '../common/splitListener';
export class SparkSqlSplitListener extends SplitListener {
    exitSingleStatement(ctx) {
        this._statementsContext.push(ctx);
    }
}
