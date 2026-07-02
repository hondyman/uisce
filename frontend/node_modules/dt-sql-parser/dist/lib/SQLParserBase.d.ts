import * as antlr from "antlr4ng";
export declare abstract class SQLParserBase<T = antlr.ParserRuleContext> extends antlr.Parser {
    constructor(input: antlr.TokenStream);
    abstract program(): T;
    caretTokenIndex: number;
    entityCollecting: boolean;
    shouldMatchEmpty(): boolean;
}
