import { DefaultErrorStrategy, Parser, RecognitionException, Token } from 'antlr4ng';
/**
 * Base on DefaultErrorStrategy.
 * The difference is that it assigns exception to the context.exception when it encounters error.
 */
export declare class ErrorStrategy extends DefaultErrorStrategy {
    recover(recognizer: Parser, e: RecognitionException): void;
    recoverInline(recognizer: Parser): Token;
}
