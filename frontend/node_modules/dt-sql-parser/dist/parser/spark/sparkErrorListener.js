import { CodeCompletionCore } from 'antlr4-c3';
import { ParseErrorListener } from '../common/parseErrorListener';
import { SparkSqlParser } from '../../lib/spark/SparkSqlParser';
export class SparkErrorListener extends ParseErrorListener {
    constructor() {
        super(...arguments);
        this.objectNames = new Map([
            [SparkSqlParser.RULE_namespaceName, 'namespace'],
            [SparkSqlParser.RULE_namespaceNameCreate, 'namespace'],
            [SparkSqlParser.RULE_tableName, 'table'],
            [SparkSqlParser.RULE_tableNameCreate, 'table'],
            [SparkSqlParser.RULE_viewName, 'view'],
            [SparkSqlParser.RULE_viewNameCreate, 'view'],
            [SparkSqlParser.RULE_functionName, 'function'],
            [SparkSqlParser.RULE_functionNameCreate, 'function'],
            [SparkSqlParser.RULE_columnName, 'column'],
            [SparkSqlParser.RULE_columnNamePath, 'column'],
            [SparkSqlParser.RULE_columnNameCreate, 'column'],
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
                    case SparkSqlParser.RULE_namespaceName:
                    case SparkSqlParser.RULE_tableName:
                    case SparkSqlParser.RULE_viewName:
                    case SparkSqlParser.RULE_functionName:
                    case SparkSqlParser.RULE_columnName:
                    case SparkSqlParser.RULE_columnNamePath: {
                        if (!result.includes(`{existing}${name}`)) {
                            result.push(`{existing}${name}`);
                        }
                        break;
                    }
                    case SparkSqlParser.RULE_namespaceNameCreate:
                    case SparkSqlParser.RULE_tableNameCreate:
                    case SparkSqlParser.RULE_functionNameCreate:
                    case SparkSqlParser.RULE_viewNameCreate:
                    case SparkSqlParser.RULE_columnNameCreate: {
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
