import { SymbolConstructor, BaseSymbol } from "./BaseSymbol.js";
/** Defines the  */
export interface IScopedSymbol extends BaseSymbol {
    /**
     * @returns A promise resolving to all direct child symbols with a scope (e.g. classes in a module).
     */
    directScopes: Promise<ScopedSymbol[]>;
    children: BaseSymbol[];
    firstChild: BaseSymbol | undefined;
    lastChild: BaseSymbol | undefined;
    clear(): void;
    /**
     * Adds the given symbol to this scope. If it belongs already to a different scope
     * it is removed from that before adding it here.
     *
     * @param symbol The symbol to add as a child.
     */
    addSymbol(symbol: BaseSymbol): void;
    /** Removes the given symbol from this scope, if it exists. */
    removeSymbol(symbol: BaseSymbol): void;
    /**
     * Asynchronously retrieves child symbols of a given type from this symbol.
     *
     * @param t The type of of the objects to return.
     *
     * @returns A promise resolving to all (nested) children of the given type.
     */
    getNestedSymbolsOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): Promise<T[]>;
    /**
     * Synchronously retrieves child symbols of a given type from this symbol.
     *
     * @param t The type of of the objects to return.
     *
     * @returns A list of all (nested) children of the given type.
     */
    getNestedSymbolsOfTypeSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): T[];
    /**
     * @param name If given only returns symbols with that name.
     *
     * @returns A promise resolving to symbols from this and all nested scopes in the order they were defined.
     */
    getAllNestedSymbols(name?: string): Promise<BaseSymbol[]>;
    /**
     * @param name If given only returns symbols with that name.
     *
     * @returns A list of all symbols from this and all nested scopes in the order they were defined.
     */
    getAllNestedSymbolsSync(name?: string): BaseSymbol[];
    /**
     * @param t The type of of the objects to return.
     *
     * @returns A promise resolving to direct children of a given type.
     */
    getSymbolsOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): Promise<T[]>;
    /**
     * TODO: add optional position dependency (only symbols defined before a given caret pos are viable).
     *
     * @param t The type of the objects to return.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A promise resolving to all symbols of the the given type, accessible from this scope (if localOnly is
     *          false), within the owning symbol table.
     */
    getAllSymbols<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): Promise<T[]>;
    /**
     * TODO: add optional position dependency (only symbols defined before a given caret pos are viable).
     *
     * @param t The type of the objects to return.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A list with all symbols of the the given type, accessible from this scope (if localOnly is
     *          false), within the owning symbol table.
     */
    getAllSymbolsSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): T[];
    /**
     * @param name The name of the symbol to resolve.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A promise resolving to the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolve(name: string, localOnly?: boolean): Promise<BaseSymbol | undefined>;
    /**
     * @param name The name of the symbol to resolve.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolveSync(name: string, localOnly?: boolean): BaseSymbol | undefined;
    /**
     * @param path The path consisting of symbol names separator by `separator`.
     * @param separator The character to separate path segments.
     *
     * @returns the symbol located at the given path through the symbol hierarchy.
     */
    symbolFromPath(path: string, separator: string): BaseSymbol | undefined;
    /**
     * @param child The child to search for.
     *
     * @returns the index of the given child symbol in the child list or -1 if it couldn't be found.
     */
    indexOfChild(child: BaseSymbol): number;
    /**
     * @param child The reference node.
     *
     * @returns the sibling symbol after the given child symbol, if one exists.
     */
    nextSiblingOf(child: BaseSymbol): BaseSymbol | undefined;
    /**
     * @param child The reference node.
     *
     * @returns the sibling symbol before the given child symbol, if one exists.
     */
    previousSiblingOf(child: BaseSymbol): BaseSymbol | undefined;
    /**
     * @param child The reference node.
     *
     * @returns the next symbol in definition order, regardless of the scope.
     */
    nextOf(child: BaseSymbol): BaseSymbol | undefined;
}
/** A symbol with a scope (so it can have child symbols). */
export declare class ScopedSymbol extends BaseSymbol implements IScopedSymbol {
    #private;
    constructor(name?: string);
    /**
     * @returns A promise resolving to all direct child symbols with a scope (e.g. classes in a module).
     */
    get directScopes(): Promise<ScopedSymbol[]>;
    get children(): BaseSymbol[];
    get firstChild(): BaseSymbol | undefined;
    get lastChild(): BaseSymbol | undefined;
    clear(): void;
    /**
     * Adds the given symbol to this scope. If it belongs already to a different scope
     * it is removed from that before adding it here.
     *
     * @param symbol The symbol to add as a child.
     */
    addSymbol(symbol: BaseSymbol): void;
    removeSymbol(symbol: BaseSymbol): void;
    /**
     * Asynchronously retrieves child symbols of a given type from this symbol.
     *
     * @param t The type of of the objects to return.
     *
     * @returns A promise resolving to all (nested) children of the given type.
     */
    getNestedSymbolsOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): Promise<T[]>;
    /**
     * Synchronously retrieves child symbols of a given type from this symbol.
     *
     * @param t The type of of the objects to return.
     *
     * @returns A list of all (nested) children of the given type.
     */
    getNestedSymbolsOfTypeSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): T[];
    /**
     * @param name If given only returns symbols with that name.
     *
     * @returns A promise resolving to symbols from this and all nested scopes in the order they were defined.
     */
    getAllNestedSymbols(name?: string): Promise<BaseSymbol[]>;
    /**
     * @param name If given only returns symbols with that name.
     *
     * @returns A list of all symbols from this and all nested scopes in the order they were defined.
     */
    getAllNestedSymbolsSync(name?: string): BaseSymbol[];
    /**
     * @param t The type of of the objects to return.
     *
     * @returns A promise resolving to direct children of a given type.
     */
    getSymbolsOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>): Promise<T[]>;
    /**
     * TODO: add optional position dependency (only symbols defined before a given caret pos are viable).
     *
     * @param t The type of the objects to return.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A promise resolving to all symbols of the the given type, accessible from this scope (if localOnly is
     *          false), within the owning symbol table.
     */
    getAllSymbols<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): Promise<T[]>;
    /**
     * TODO: add optional position dependency (only symbols defined before a given caret pos are viable).
     *
     * @param t The type of the objects to return.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A list with all symbols of the the given type, accessible from this scope (if localOnly is
     *          false), within the owning symbol table.
     */
    getAllSymbolsSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): T[];
    /**
     * @param name The name of the symbol to resolve.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns A promise resolving to the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolve(name: string, localOnly?: boolean): Promise<BaseSymbol | undefined>;
    /**
     * @param name The name of the symbol to resolve.
     * @param localOnly If true only child symbols are returned, otherwise also symbols from the parent of this symbol
     *                  (recursively).
     *
     * @returns the first symbol with a given name, in the order of appearance in this scope
     *          or any of the parent scopes (conditionally).
     */
    resolveSync(name: string, localOnly?: boolean): BaseSymbol | undefined;
    /**
     * @param path The path consisting of symbol names separator by `separator`.
     * @param separator The character to separate path segments.
     *
     * @returns the symbol located at the given path through the symbol hierarchy.
     */
    symbolFromPath(path: string, separator?: string): BaseSymbol | undefined;
    /**
     * @param child The child to search for.
     *
     * @returns the index of the given child symbol in the child list or -1 if it couldn't be found.
     */
    indexOfChild(child: BaseSymbol): number;
    /**
     * @param child The reference node.
     *
     * @returns the sibling symbol after the given child symbol, if one exists.
     */
    nextSiblingOf(child: BaseSymbol): BaseSymbol | undefined;
    /**
     * @param child The reference node.
     *
     * @returns the sibling symbol before the given child symbol, if one exists.
     */
    previousSiblingOf(child: BaseSymbol): BaseSymbol | undefined;
    /**
     * @param child The reference node.
     *
     * @returns the next symbol in definition order, regardless of the scope.
     */
    nextOf(child: BaseSymbol): BaseSymbol | undefined;
    private isNamespace;
}
