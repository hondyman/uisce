import { Token, LexerNoViableAltException, InputMismatchException, NoViableAltException, } from 'antlr4ng';
import { transform } from './transform';
export class ParseErrorListener {
    get locale() {
        return this.parserContext.locale;
    }
    constructor(errorListener, parserContext, preferredRules) {
        this.parserContext = parserContext;
        this.preferredRules = preferredRules;
        this._errorListener = errorListener;
    }
    reportAmbiguity() { }
    reportAttemptingFullContext() { }
    reportContextSensitivity() { }
    syntaxError(recognizer, offendingSymbol, line, charPositionInLine, msg, e) {
        var _a;
        let message = '';
        // If not undefined then offendingSymbol is of type Token.
        if (offendingSymbol) {
            let token = offendingSymbol;
            const parser = recognizer;
            // judge token is EOF
            const isEof = token.type === Token.EOF;
            if (isEof) {
                token = parser.tokenStream.get(token.tokenIndex - 1);
            }
            const wrongText = (_a = token.text) !== null && _a !== void 0 ? _a : '';
            const isInComplete = isEof && wrongText !== ' ';
            const expectedText = isInComplete ? '' : this.getExpectedText(parser, token);
            if (!e) {
                // handle missing or unwanted tokens.
                message = msg;
                if (msg.includes('extraneous')) {
                    message = `'${wrongText}' {noValidPosition}${expectedText.length ? `{expecting}${expectedText}` : ''}`;
                }
                if (msg.includes('missing')) {
                    const regex = /missing\s+'([^']+)'/;
                    const match = msg.match(regex);
                    message = `{missing}`;
                    if (match) {
                        const missKeyword = match[1];
                        message += `'${missKeyword}'`;
                    }
                    else {
                        message += `{keyword}`;
                    }
                    message += `{at}'${wrongText}'`;
                }
            }
            else {
                // handle mismatch exception or no viable alt exception
                if (e instanceof InputMismatchException || e instanceof NoViableAltException) {
                    if (isEof) {
                        message = `{stmtInComplete}`;
                    }
                    else {
                        message = `'${wrongText}' {noValidPosition}`;
                    }
                    if (expectedText.length > 0) {
                        message += `{expecting}${expectedText}`;
                    }
                }
                else {
                    message = msg;
                }
            }
        }
        else {
            // No offending symbol, which indicates this is a lexer error.
            if (e instanceof LexerNoViableAltException) {
                const lexer = recognizer;
                const input = lexer.inputStream;
                let text = lexer.getErrorDisplay(input.getText(lexer._tokenStartCharIndex, input.index));
                switch (text[0]) {
                    case '/':
                        message = '{unfinishedMultilineComment}';
                        break;
                    case '"':
                        message = '{unfinishedDoubleQuoted}';
                        break;
                    case "'":
                        message = '{unfinishedSingleQuoted}';
                        break;
                    case '`':
                        message = '{unfinishedTickQuoted}';
                        break;
                    default:
                        message = '"' + text + '" {noValidInput}';
                        break;
                }
            }
        }
        message = transform(message, this.locale);
        let endCol = charPositionInLine + 1;
        if (offendingSymbol && offendingSymbol.text !== null) {
            endCol = charPositionInLine + offendingSymbol.text.length;
        }
        if (this._errorListener) {
            this._errorListener({
                startLine: line,
                endLine: line,
                startColumn: charPositionInLine + 1,
                endColumn: endCol + 1,
                message,
            }, {
                e,
                line,
                msg,
                recognizer,
                offendingSymbol,
                charPositionInLine,
            });
        }
    }
}
