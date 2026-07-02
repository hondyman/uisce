import { CodeCompletionCore } from 'antlr4-c3';
import { TrinoSqlParser } from '../../lib/trino/TrinoSqlParser';
import { ParseErrorListener } from '../common/parseErrorListener';
export class TrinoErrorListener extends ParseErrorListener {
    constructor() {
        super(...arguments);
        this.objectNames = new Map([
            [TrinoSqlParser.RULE_catalogRef, 'catalog'],
            [TrinoSqlParser.RULE_catalogNameCreate, 'catalog'],
            [TrinoSqlParser.RULE_tableRef, 'table'],
            [TrinoSqlParser.RULE_tableNameCreate, 'table'],
            [TrinoSqlParser.RULE_viewRef, 'view'],
            [TrinoSqlParser.RULE_viewNameCreate, 'view'],
            [TrinoSqlParser.RULE_schemaRef, 'schema'],
            [TrinoSqlParser.RULE_schemaNameCreate, 'schema'],
            [TrinoSqlParser.RULE_functionName, 'function'],
            [TrinoSqlParser.RULE_columnRef, 'column'],
            [TrinoSqlParser.RULE_columnName, 'column'],
            [TrinoSqlParser.RULE_columnNameCreate, 'column'],
        ]);
    }
    getExpectedText(parser, token) {
        var _a;
        let expectedText = '';
        const input = this.parserContext.getParsedInput();
        /**
         * Get the program context.
         * When called error listener, `this._parseTree` is still `undefined`,
         * so we can't use cached parseTree in `getMinimumParserInfo`
         */
        let currentContext = (_a = parser.context) !== null && _a !== void 0 ? _a : undefined;
        while (currentContext === null || currentContext === void 0 ? void 0 : currentContext.parent) {
            currentContext = currentContext.parent;
        }
        const parserInfo = this.parserContext.getMinimumParserInfo(input, token.tokenIndex, currentContext);
        if (!parserInfo)
            return '';
        const { parser: c3Parser, newTokenIndex, parseTree: c3Context } = parserInfo;
        const core = new CodeCompletionCore(c3Parser);
        core.preferredRules = this.preferredRules;
        const candidates = core.collectCandidates(newTokenIndex, c3Context);
        if (candidates.rules.size) {
            const result = [];
            // get expectedText as collect rules first
            for (const candidate of candidates.rules) {
                const [ruleType] = candidate;
                const name = this.objectNames.get(ruleType);
                switch (ruleType) {
                    case TrinoSqlParser.RULE_catalogRef:
                    case TrinoSqlParser.RULE_schemaRef:
                    case TrinoSqlParser.RULE_tableRef:
                    case TrinoSqlParser.RULE_viewRef:
                    case TrinoSqlParser.RULE_functionName:
                    case TrinoSqlParser.RULE_columnRef:
                    case TrinoSqlParser.RULE_columnName: {
                        if (!result.includes(`{existing}${name}`)) {
                            result.push(`{existing}${name}`);
                        }
                        break;
                    }
                    case TrinoSqlParser.RULE_catalogNameCreate:
                    case TrinoSqlParser.RULE_tableNameCreate:
                    case TrinoSqlParser.RULE_schemaNameCreate:
                    case TrinoSqlParser.RULE_viewNameCreate:
                    case TrinoSqlParser.RULE_tableNameCreate: {
                        if (!result.includes(`{new}${name}`)) {
                            result.push(`{new}${name}`);
                        }
                        break;
                    }
                }
            }
            expectedText = result.join('{or}');
        }
        if (candidates.tokens.size) {
            expectedText += expectedText ? '{orKeyword}' : '{keyword}';
        }
        return expectedText;
    }
}
