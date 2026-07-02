import { Token } from 'antlr4ng';
import { CaretPosition } from './types';
/**
 * find token index via caret position (cursor position)
 * @param caretPosition
 * @param allTokens all the tokens
 * @returns caretTokenIndex
 */
export declare function findCaretTokenIndex(caretPosition: CaretPosition, allTokens: Token[]): number | undefined;
