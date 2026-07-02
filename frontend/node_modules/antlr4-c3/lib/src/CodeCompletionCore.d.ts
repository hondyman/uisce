import { Parser, ParserRuleContext } from "antlr4ng";
export type TokenList = number[];
export type RuleList = number[];
export interface ICandidateRule {
    startTokenIndex: number;
    ruleList: RuleList;
}
export interface IRuleWithStartToken {
    startTokenIndex: number;
    ruleIndex: number;
}
export type RuleWithStartTokenList = IRuleWithStartToken[];
/**
 * All the candidates which have been found. Tokens and rules are separated.
 * Token entries include a list of tokens that directly follow them (see also the "following" member in the
 * FollowSetWithPath class).
 * Rule entries include the index of the starting token within the evaluated rule, along with a call stack of rules
 * found during evaluation.
 */
export declare class CandidatesCollection {
    tokens: Map<number, TokenList>;
    rules: Map<number, ICandidateRule>;
}
/** The main class for doing the collection process. */
export declare class CodeCompletionCore {
    private static followSetsByATN;
    private static atnStateTypeMap;
    /** Not dependent on showDebugOutput. Prints the collected rules + tokens to terminal. */
    showResult: boolean;
    /** Enables printing ATN state info to terminal. */
    showDebugOutput: boolean;
    /** Only relevant when showDebugOutput is true. Enables transition printing for a state. */
    debugOutputWithTransitions: boolean;
    /** Also depends on showDebugOutput. Enables call stack printing for each rule recursion. */
    showRuleStack: boolean;
    /**
     * Tailoring of the result:
     * Tokens which should not appear in the candidates set.
     */
    ignoredTokens: Set<number>;
    /**
     * Rules which replace any candidate token they contain.
     * This allows to return descriptive rules (e.g. className, instead of ID/identifier).
     */
    preferredRules: Set<number>;
    /**
     * Specify if preferred rules should translated top-down (higher index rule returns first) or
     * bottom-up (lower index rule returns first).
     */
    translateRulesTopDown: boolean;
    private parser;
    private atn;
    private vocabulary;
    private ruleNames;
    private tokens;
    private precedenceStack;
    private tokenStartIndex;
    private statesProcessed;
    /**
     * A mapping of rule index + token stream position to end token positions.
     * A rule which has been visited before with the same input position will always produce the same output positions.
     */
    private shortcutMap;
    /** The collected candidates (rules and tokens). */
    private candidates;
    constructor(parser: Parser);
    /**
     * This is the main entry point. The caret token index specifies the token stream index for the token which
     * currently covers the caret (or any other position you want to get code completion candidates for).
     * Optionally you can pass in a parser rule context which limits the ATN walk to only that or called rules.
     * This can significantly speed up the retrieval process but might miss some candidates (if they are outside of
     * the given context).
     *
     * @param caretTokenIndex The index of the token at the caret position.
     * @param context An option parser rule context to limit the search space.
     * @returns The collection of completion candidates.
     */
    collectCandidates(caretTokenIndex: number, context?: ParserRuleContext): CandidatesCollection;
    /**
     * Checks if the predicate associated with the given transition evaluates to true.
     *
     * @param transition The transition to check.
     * @returns the evaluation result of the predicate.
     */
    private checkPredicate;
    /**
     * Walks the rule chain upwards or downwards (depending on translateRulesTopDown) to see if that matches any of the
     * preferred rules. If found, that rule is added to the collection candidates and true is returned.
     *
     * @param ruleWithStartTokenList The list to convert.
     * @returns true if any of the stack entries was converted.
     */
    private translateStackToRuleIndex;
    /**
     * Given the index of a rule from a rule chain, check if that matches any of the preferred rules. If it matches,
     * that rule is added to the collection candidates and true is returned.
     *
     * @param i The rule index.
     * @param ruleWithStartTokenList The list to check.
     * @returns true if the specified rule is in the list of preferred rules.
     */
    private translateToRuleIndex;
    /**
     * This method follows the given transition and collects all symbols within the same rule that directly follow it
     * without intermediate transitions to other rules and only if there is a single symbol for a transition.
     *
     * @param transition The transition from which to start.
     * @returns A list of toke types.
     */
    private getFollowingTokens;
    /**
     * Entry point for the recursive follow set collection function.
     *
     * @param start Start state.
     * @param stop Stop state.
     * @returns Follow sets.
     */
    private determineFollowSets;
    /**
     * Collects possible tokens which could be matched following the given ATN state. This is essentially the same
     * algorithm as used in the LL1Analyzer class, but here we consider predicates also and use no parser rule context.
     *
     * @param s The state to continue from.
     * @param stopState The state which ends the collection routine.
     * @param followSets A pass through parameter to add found sets to.
     * @param stateStack A stack to avoid endless recursions.
     * @param ruleStack The current rule stack.
     * @returns true if the follow sets is exhaustive, i.e. we terminated before the rule end was reached, so no
     * subsequent rules could add tokens
     */
    private collectFollowSets;
    /**
     * Walks the ATN for a single rule only. It returns the token stream position for each path that could be matched
     * in this rule.
     * The result can be empty in case we hit only non-epsilon transitions that didn't match the current input or if we
     * hit the caret position.
     *
     * @param startState The start state.
     * @param tokenListIndex The token index we are currently at.
     * @param callStack The stack that indicates where in the ATN we are currently.
     * @param precedence The current precedence level.
     * @param indentation A value to determine the current indentation when doing debug prints.
     * @returns the set of token stream indexes (which depend on the ways that had to be taken).
     */
    private processRule;
    private generateBaseDescription;
    private printDescription;
    private printRuleState;
}
