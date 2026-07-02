import { type ParseTree } from "antlr4ng";
import { type IScopedSymbol } from "./ScopedSymbol.js";
import { type ISymbolTable } from "./SymbolTable.js";
import { MemberVisibility, Modifier } from "./types.js";
/**
 * The root of the symbol table class hierarchy: a symbol can be any manageable entity (like a block), not only
 * things like variables or classes.
 * We are using a class hierarchy here, instead of an enum or similar, to allow for easy extension and certain
 * symbols can so provide additional APIs for simpler access to their sub elements, if needed.
 */
export declare class BaseSymbol {
    #private;
    /** The name of the symbol or empty if anonymous. */
    name: string;
    /** Reference to the parse tree which contains this symbol. */
    context?: ParseTree;
    readonly modifiers: Set<Modifier>;
    visibility: MemberVisibility;
    constructor(name?: string);
    get parent(): IScopedSymbol | undefined;
    get firstSibling(): BaseSymbol | undefined;
    /**
     * @returns the symbol before this symbol in its scope.
     */
    get previousSibling(): BaseSymbol | undefined;
    /**
     * @returns the symbol following this symbol in its scope.
     */
    get nextSibling(): BaseSymbol | undefined;
    get lastSibling(): BaseSymbol | undefined;
    /**
     * @returns the next symbol in definition order, regardless of the scope.
     */
    get next(): BaseSymbol | undefined;
    /**
     * @returns the outermost entity (below the symbol table) that holds us.
     */
    get root(): BaseSymbol | undefined;
    /**
     * @returns the symbol table we belong too or undefined if we are not yet assigned.
     */
    get symbolTable(): ISymbolTable | undefined;
    /**
     * @returns the list of symbols from this one up to root.
     */
    get symbolPath(): BaseSymbol[];
    /**
     * This is rather an internal method and should rarely be used by external code.
     *
     * @param parent The new parent to use.
     */
    setParent(parent?: IScopedSymbol): void;
    /**
     * Remove this symbol from its parent scope.
     */
    removeFromParent(): void;
    /**
     * Asynchronously looks up a symbol with a given name, in a bottom-up manner.
     *
     * @param name The name of the symbol to find.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A promise resolving to the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolve(name: string, localOnly?: boolean): Promise<BaseSymbol | undefined>;
    /**
     * Synchronously looks up a symbol with a given name, in a bottom-up manner.
     *
     * @param name The name of the symbol to find.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolveSync(name: string, localOnly?: boolean): BaseSymbol | undefined;
    /**
     * @param t The type of objects to return.
     *
     * @returns the next enclosing parent of the given type.
     */
    getParentOfType<T extends BaseSymbol>(t: SymbolConstructor<T, unknown[]>): T | undefined;
    /**
     * Creates a qualified identifier from this symbol and its parent.
     * If `full` is true then all parents are traversed in addition to this instance.
     *
     * @param separator The string to be used between the parts.
     * @param full A flag indicating if the full path is to be returned.
     * @param includeAnonymous Use a special string for empty scope names.
     *
     * @returns the constructed qualified identifier.
     */
    qualifiedName(separator?: string, full?: boolean, includeAnonymous?: boolean): string;
    /**
     * Type guard to check for ISymbolTable.
     *
     * @param candidate The object to check.
     *
     * @returns true if the object is a symbol table.
     */
    private isSymbolTable;
}
/** The type of constructors for symbols. Used mostly for factory and lookup functions. */
export type SymbolConstructor<T extends BaseSymbol, Args extends unknown[]> = new (...args: Args) => T;
