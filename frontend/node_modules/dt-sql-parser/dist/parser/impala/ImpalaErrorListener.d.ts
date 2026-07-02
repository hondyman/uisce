import { ParseErrorListener } from '../common/parseErrorListener';
import { Parser, Token } from 'antlr4ng';
export declare class ImpalaErrorListener extends ParseErrorListener {
    private objectNames;
    getExpectedText(parser: Parser, token: Token): string;
}
