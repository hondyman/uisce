import { ErrorNode, ParserRuleContext, Token } from 'antlr4ng';
import { findCaretTokenIndex } from '../common/findCaretTokenIndex';
import { SqlSplitStrategy, } from '../common/types';
import { SQL_SPLIT_SYMBOL_TEXT } from './basicSQL';
class SemanticContextCollector {
    constructor(_input, caretPosition, allTokens, options) {
        this.options = {
            sqlSplitStrategy: SqlSplitStrategy.STRICT,
        };
        this._allTokens = [];
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
        this._isStatementBeginning = false;
        // If caretPosition token is whiteSpace, tokenIndex may be undefined.
        const tokenIndex = findCaretTokenIndex(caretPosition, allTokens);
        if (tokenIndex !== undefined) {
            this._tokenIndex = tokenIndex;
        }
        this._allTokens = allTokens;
        this.options = Object.assign(Object.assign({}, this.options), options);
        if (allTokens === null || allTokens === void 0 ? void 0 : allTokens.length) {
            let i = tokenIndex ? tokenIndex - 1 : allTokens.length - 1;
            /**
             * Link to @case4 and @case5
             * Find the previous unhidden token.
             * If can't find tokenIndex or current token is whiteSpace at caretPosition,
             * prevTokenIndex is useful to help us determine if it is beginning of statement.
             */
            while (i >= 0) {
                if (allTokens[i].channel !== Token.HIDDEN_CHANNEL &&
                    (allTokens[i].line < caretPosition.lineNumber ||
                        (allTokens[i].line === caretPosition.lineNumber &&
                            allTokens[i].column < caretPosition.column))) {
                    this._prevTokenIndex = allTokens[i].tokenIndex;
                    break;
                }
                i--;
            }
            /**
             * We can directly conclude beginning of statement semantics when current token is
             * the first token of tokenStream or the previous token is semicolon
             */
            if (tokenIndex === 0 ||
                i === -1 ||
                (this._prevTokenIndex &&
                    this._allTokens[this._prevTokenIndex].text === SQL_SPLIT_SYMBOL_TEXT)) {
                this._isStatementBeginning = true;
            }
        }
    }
    get semanticContext() {
        return {
            isStatementBeginning: this._isStatementBeginning,
        };
    }
    prevStatementHasError(node) {
        let parent = node.parent;
        if (!parent)
            return false;
        const currentNodeIndex = parent.children.findIndex((child) => child === node);
        if (currentNodeIndex <= 0)
            return false;
        for (let i = currentNodeIndex - 1; i >= 0; i--) {
            const prevNode = parent.children[i];
            if (prevNode instanceof ErrorNode ||
                (prevNode instanceof ParserRuleContext && prevNode.exception !== null))
                return true;
        }
        return false;
    }
    /**
     * Most root rule is `program`.
     */
    isRootRule(node) {
        return node instanceof ParserRuleContext && (node === null || node === void 0 ? void 0 : node.parent) === null;
    }
    /**
     * link to @case4
     * It should be called in each language's own `enterStatement`.
     */
    visitStatement(ctx) {
        var _a, _b, _c;
        if (this.options.sqlSplitStrategy === SqlSplitStrategy.STRICT)
            return;
        const isWhiteSpaceToken = this._tokenIndex === undefined ||
            ((_a = this._allTokens[this._tokenIndex]) === null || _a === void 0 ? void 0 : _a.type) === this.getWhiteSpaceRuleType() ||
            // PostgreSQL whiteSpace not inlcudes '\n' symbol
            ((_b = this._allTokens[this._tokenIndex]) === null || _b === void 0 ? void 0 : _b.text) === '\n';
        const isPrevTokenEndOfStatement = this._prevTokenIndex && ((_c = ctx.stop) === null || _c === void 0 ? void 0 : _c.tokenIndex) === this._prevTokenIndex;
        if (isWhiteSpaceToken && isPrevTokenEndOfStatement && ctx.exception === null) {
            this._isStatementBeginning = !this.prevStatementHasError(ctx)
                ? true
                : this._isStatementBeginning;
        }
    }
    /**
     * Uncomplete keyword will be error node
     */
    visitErrorNode(node) {
        var _a, _b, _c, _d;
        if (node.symbol.tokenIndex !== this._tokenIndex ||
            this._isStatementBeginning ||
            this.options.sqlSplitStrategy === SqlSplitStrategy.STRICT)
            return;
        let parent = node.parent;
        let currentNode = node;
        /**
         * Link to @case2
         * The error node is a direct child node of the program node
         */
        if (this.isRootRule(parent)) {
            this._isStatementBeginning = !this.prevStatementHasError(currentNode);
            return;
        }
        /**
         * Link to @case3
         * Error node must be the first leaf node of the statement parse tree.
         **/
        while (parent !== null && parent.ruleIndex !== this.getStatementRuleType()) {
            if (((_a = parent.children) === null || _a === void 0 ? void 0 : _a[0]) !== currentNode) {
                this._isStatementBeginning = false;
                return;
            }
            currentNode = parent;
            parent = currentNode.parent;
        }
        let isStatementBeginning = true;
        /**
         * Link to @case1
         * Previous statement must have no exception
         */
        if ((parent === null || parent === void 0 ? void 0 : parent.ruleIndex) === this.getStatementRuleType()) {
            const programRule = parent.parent;
            const currentStatementRuleIndex = ((_b = programRule === null || programRule === void 0 ? void 0 : programRule.children) === null || _b === void 0 ? void 0 : _b.findIndex((node) => node === parent)) || -1;
            if (currentStatementRuleIndex > 0) {
                /**
                 * When you typed a keyword and doesn't match any rule, you will get a EOF error,
                 * For example, just typed 'CREATE', 'INSERT'.
                 */
                const isStatementEOF = ((_d = (_c = parent.exception) === null || _c === void 0 ? void 0 : _c.offendingToken) === null || _d === void 0 ? void 0 : _d.text) === '<EOF>';
                isStatementBeginning =
                    this.prevStatementHasError(parent) && !isStatementEOF
                        ? false
                        : isStatementBeginning;
            }
        }
        this._isStatementBeginning = isStatementBeginning;
    }
    visitTerminal(node) {
        var _a, _b;
        if (node.symbol.tokenIndex !== this._tokenIndex ||
            this._isStatementBeginning ||
            this.options.sqlSplitStrategy === SqlSplitStrategy.STRICT)
            return;
        let currentNode = node;
        let parent = node.parent;
        /**
         * Link to @case3
         * Current terminal node must be the first leaf node of the statement parse tree.
         **/
        while (parent !== null && parent.ruleIndex !== this.getStatementRuleType()) {
            if (((_a = parent.children) === null || _a === void 0 ? void 0 : _a[0]) !== currentNode) {
                this._isStatementBeginning = false;
                return;
            }
            currentNode = parent;
            parent = currentNode.parent;
        }
        let isStatementBeginning = true;
        /**
         * Link to @case1
         * Previous statement must have no exception
         */
        if ((parent === null || parent === void 0 ? void 0 : parent.ruleIndex) === this.getStatementRuleType()) {
            const programRule = parent.parent;
            const currentStatementRuleIndex = ((_b = programRule === null || programRule === void 0 ? void 0 : programRule.children) === null || _b === void 0 ? void 0 : _b.findIndex((node) => node === parent)) || -1;
            if (currentStatementRuleIndex > 0) {
                isStatementBeginning = this.prevStatementHasError(parent)
                    ? false
                    : isStatementBeginning;
            }
        }
        this._isStatementBeginning = isStatementBeginning;
    }
    enterEveryRule(_node) { }
    exitEveryRule(_node) { }
}
export default SemanticContextCollector;
