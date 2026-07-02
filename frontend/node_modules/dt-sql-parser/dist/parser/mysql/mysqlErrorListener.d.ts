import { ParseErrorListener } from '../common/parseErrorListener';
import { Parser, Token } from 'antlr4ng';
export declare class MysqlErrorListener extends ParseErrorListener {
    private objectNames;
    getExpectedText(parser: Parser, token: Token): string;
}
