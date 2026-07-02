import { ErrorNode, ParserRuleContext, TerminalNode, Token } from 'antlr4ng';
import { CaretPosition, SemanticCollectOptions, SemanticContext } from '../common/types';
declare abstract class SemanticContextCollector {
    constructor(_input: string, caretPosition: CaretPosition, allTokens: Token[], options?: SemanticCollectOptions);
    readonly options: SemanticCollectOptions;
    private _tokenIndex;
    private _allTokens;
    /**
     * If current caret position is in a beginning of statement semantics, it needs to follow some cases:
     * @case1 there is no statement node with an error before the current statement in the parse tree;
     *
     * @case2 if it is an uncomplete keyword, it will be parsed as an `ErrorNode`
     * and need be a direct child node of `program`;
     *
     * @case3 if it is a complete keyword, the parsed TerminalNode or ErrorNode should be
     * the first leaf node of current statement rule;
     *
     * @case4 if it is whiteSpace in caret position, we can't visit it in antlr4 listener,
     * so we find the first unhidden token before the whiteSpace token, and the unhidden token
     * should be the last leaf node of statement its belongs to;
     *
     * @case5 if the previous token is split symbol like `;`, ignore case1 and forcefully judged as beginning of statement.
     */
    private _isStatementBeginning;
    /**
     * Prev tokenIndex that not white space before current tokenIndex or caret position
     */
    private _prevTokenIndex;
    get semanticContext(): SemanticContext;
    abstract getWhiteSpaceRuleType(): number;
    abstract getStatementRuleType(): number;
    private prevStatementHasError;
    /**
     * Most root rule is `program`.
     */
    private isRootRule;
    /**
     * link to @case4
     * It should be called in each language's own `enterStatement`.
     */
    protected visitStatement(ctx: ParserRuleContext): void;
    /**
     * Uncomplete keyword will be error node
     */
    visitErrorNode(node: ErrorNode): void;
    visitTerminal(node: TerminalNode): void;
    enterEveryRule(_node: ParserRuleContext): void;
    exitEveryRule(_node: ParserRuleContext): void;
}
export default SemanticContextCollector;
