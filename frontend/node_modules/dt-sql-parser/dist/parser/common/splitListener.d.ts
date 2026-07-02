export declare abstract class SplitListener<T> {
    protected _statementsContext: T[];
    visitTerminal(): void;
    visitErrorNode(): void;
    enterEveryRule(): void;
    exitEveryRule(): void;
    get statementsContext(): T[];
}
