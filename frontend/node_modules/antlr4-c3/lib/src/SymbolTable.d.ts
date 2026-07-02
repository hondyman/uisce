import { ParseTree } from "antlr4ng";
import { ISymbolTableOptions } from "./types.js";
import { SymbolConstructor, BaseSymbol } from "./BaseSymbol.js";
import { IScopedSymbol, ScopedSymbol } from "./ScopedSymbol.js";
import { NamespaceSymbol } from "./NamespaceSymbol.js";
export interface ISymbolTable extends IScopedSymbol {
    options: ISymbolTableOptions;
    /**
     * @returns instance information, mostly relevant for unit testing.
     */
    info: {
        dependencyCount: number;
        symbolCount: number;
    };
    clear(): void;
    addDependencies(...tables: SymbolTable[]): void;
    removeDependency(table: SymbolTable): void;
    addNewSymbolOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, parent: ScopedSymbol | undefined, ...args: Args): T;
    /**
     * Asynchronously adds a new namespace to the symbol table or the given parent. The path parameter specifies a
     * single namespace name or a chain of namespaces (which can be e.g. "outer.intermittent.inner.final").
     * If any of the parent namespaces is missing they are created implicitly. The final part must not exist however
     * or you'll get a duplicate symbol error.
     *
     * @param parent The parent to add the namespace to.
     * @param path The namespace path.
     * @param delimiter The delimiter used in the path.
     *
     * @returns The new symbol.
     */
    addNewNamespaceFromPath(parent: ScopedSymbol | undefined, path: string, delimiter?: string): Promise<NamespaceSymbol>;
    /**
     * Synchronously adds a new namespace to the symbol table or the given parent. The path parameter specifies a
     * single namespace name or a chain of namespaces (which can be e.g. "outer.intermittent.inner.final").
     * If any of the parent namespaces is missing they are created implicitly. The final part must not exist however
     * or you'll get a duplicate symbol error.
     *
     * @param parent The parent to add the namespace to.
     * @param path The namespace path.
     * @param delimiter The delimiter used in the path.
     *
     * @returns The new symbol.
     */
    addNewNamespaceFromPathSync(parent: ScopedSymbol | undefined, path: string, delimiter?: string): NamespaceSymbol;
    /**
     * Asynchronously returns all symbols from this scope (and optionally those from dependencies) of a specific type.
     *
     * @param t The type of the symbols to return.
     * @param localOnly If true do not search dependencies.
     *
     * @returns A promise which resolves when all symbols are collected.
     */
    getAllSymbols<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly: boolean): Promise<T[]>;
    /**
     * Synchronously returns all symbols from this scope (and optionally those from dependencies) of a specific type.
     *
     * @param t The type of the symbols to return.
     * @param localOnly If true do not search dependencies.
     *
     * @returns A list with all symbols.
     */
    getAllSymbolsSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly: boolean): T[];
    /**
     * Asynchronously looks for a symbol which is connected with a given parse tree context.
     *
     * @param context The context to search for.
     *
     * @returns A promise resolving to the found symbol or undefined.
     */
    symbolWithContext(context: ParseTree): Promise<BaseSymbol | undefined>;
    /**
     * Synchronously looks for a symbol which is connected with a given parse tree context.
     *
     * @param context The context to search for.
     *
     * @returns The found symbol or undefined.
     */
    symbolWithContextSync(context: ParseTree): BaseSymbol | undefined;
    /**
     * Asynchronously resolves a name to a symbol.
     *
     * @param name The name of the symbol to find.
     * @param localOnly A flag indicating if only this symbol table should be used or also its dependencies.
     *
     * @returns A promise resolving to the found symbol or undefined.
     */
    resolve(name: string, localOnly: boolean): Promise<BaseSymbol | undefined>;
    /**
     * Synchronously resolves a name to a symbol.
     *
     * @param name The name of the symbol to find.
     * @param localOnly A flag indicating if only this symbol table should be used or also its dependencies.
     *
     * @returns The found symbol or undefined.
     */
    resolveSync(name: string, localOnly: boolean): BaseSymbol | undefined;
}
/** The main class managing all the symbols for a top level entity like a file, library or similar. */
export declare class SymbolTable extends ScopedSymbol implements ISymbolTable {
    readonly options: ISymbolTableOptions;
    /**  Other symbol information available to this instance. */
    protected dependencies: Set<SymbolTable>;
    constructor(name: string, options: ISymbolTableOptions);
    get info(): {
        dependencyCount: number;
        symbolCount: number;
    };
    clear(): void;
    addDependencies(...tables: SymbolTable[]): void;
    removeDependency(table: SymbolTable): void;
    addNewSymbolOfType<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, parent: ScopedSymbol | undefined, ...args: Args): T;
    addNewNamespaceFromPath(parent: ScopedSymbol | undefined, path: string, delimiter?: string): Promise<NamespaceSymbol>;
    addNewNamespaceFromPathSync(parent: ScopedSymbol | undefined, path: string, delimiter?: string): NamespaceSymbol;
    getAllSymbols<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): Promise<T[]>;
    getAllSymbolsSync<T extends BaseSymbol, Args extends unknown[]>(t: SymbolConstructor<T, Args>, localOnly?: boolean): T[];
    symbolWithContext(context: ParseTree): Promise<BaseSymbol | undefined>;
    symbolWithContextSync(context: ParseTree): BaseSymbol | undefined;
    resolve(name: string, localOnly?: boolean): Promise<BaseSymbol | undefined>;
    resolveSync(name: string, localOnly?: boolean): BaseSymbol | undefined;
}
