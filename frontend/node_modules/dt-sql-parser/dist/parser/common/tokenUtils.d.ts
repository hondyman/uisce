/**
 * Utility function for processing SQL tokens and generating keyword suggestions
 */
import { Parser } from 'antlr4ng';
import { CandidatesCollection } from 'antlr4-c3';
/**
 * Process token candidates and generate a list of keyword suggestions
 * @param parser SQL parser instance
 * @param tokens token candidates
 * @returns list of keyword suggestions
 */
export declare function processTokenCandidates(parser: Parser, tokens: CandidatesCollection['tokens']): string[];
