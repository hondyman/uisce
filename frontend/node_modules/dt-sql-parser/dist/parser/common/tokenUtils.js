/**
 * Utility function for processing SQL tokens and generating keyword suggestions
 */
/**
 * Process token candidates and generate a list of keyword suggestions
 * @param parser SQL parser instance
 * @param tokens token candidates
 * @returns list of keyword suggestions
 */
export function processTokenCandidates(parser, tokens) {
    const keywords = [];
    const cleanDisplayName = (displayName) => {
        return displayName && displayName.startsWith("'") && displayName.endsWith("'")
            ? displayName.slice(1, -1)
            : displayName || '';
    };
    const isKeywordToken = (token) => {
        const symbolicName = parser.vocabulary.getSymbolicName(token);
        return Boolean(symbolicName === null || symbolicName === void 0 ? void 0 : symbolicName.startsWith('KW_'));
    };
    for (const [token, followSets] of tokens) {
        const displayName = parser.vocabulary.getDisplayName(token);
        if (!displayName || !isKeywordToken(token))
            continue;
        const keyword = cleanDisplayName(displayName);
        keywords.push(keyword);
        if (followSets.length && followSets.every((s) => isKeywordToken(s))) {
            const followKeywords = followSets
                .map((s) => cleanDisplayName(parser.vocabulary.getDisplayName(s)))
                .filter(Boolean);
            if (followKeywords.length) {
                const combinedKeyword = [keyword, ...followKeywords].join(' ');
                keywords.push(combinedKeyword);
            }
        }
    }
    return keywords;
}
