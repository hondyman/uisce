import { SplitListener } from '../common/splitListener';
export class ImpalaSqlSplitListener extends SplitListener {
    exitSingleStatement(ctx) {
        this._statementsContext.push(ctx);
    }
}
