import { Parser, Token } from 'antlr4ng';
import { ParseErrorListener } from '../common/parseErrorListener';
export declare class TrinoErrorListener extends ParseErrorListener {
    private objectNames;
    getExpectedText(parser: Parser, token: Token): string;
}
