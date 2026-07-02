import { Token, Recognizer, ANTLRErrorListener, RecognitionException, ATNSimulator, Parser } from 'antlr4ng';
import { BasicSQL } from './basicSQL';
/**
 * Converted from {@link SyntaxError}.
 */
export interface ParseError {
    /** start at 1 */
    readonly startLine: number;
    /** end at ..n */
    readonly endLine: number;
    /** start at 1 */
    readonly startColumn: number;
    /** end at ..n + 1 */
    readonly endColumn: number;
    readonly message: string;
}
/**
 * The type of error resulting from lexical parsing and parsing.
 */
export interface SyntaxError {
    readonly recognizer: Recognizer<ATNSimulator>;
    readonly offendingSymbol: Token | null;
    readonly line: number;
    readonly charPositionInLine: number;
    readonly msg: string;
    readonly e: RecognitionException;
}
/**
 * ErrorListener will be invoked when it encounters a parsing error.
 * Includes lexical errors and parsing errors.
 */
export type ErrorListener = (parseError: ParseError, originalError: SyntaxError) => void;
export declare abstract class ParseErrorListener implements ANTLRErrorListener {
    private _errorListener;
    protected preferredRules: Set<number>;
    protected get locale(): import("./types").LOCALE_TYPE;
    protected parserContext: BasicSQL;
    constructor(errorListener: ErrorListener, parserContext: BasicSQL, preferredRules: Set<number>);
    reportAmbiguity(): void;
    reportAttemptingFullContext(): void;
    reportContextSensitivity(): void;
    protected abstract getExpectedText(parser: Parser, token: Token): string;
    syntaxError(recognizer: Recognizer<ATNSimulator>, offendingSymbol: Token | null, line: number, charPositionInLine: number, msg: string, e: RecognitionException): void;
}
