import { IScopedSymbol, ScopedSymbol } from "./ScopedSymbol.js";
export interface INamespaceSymbol extends IScopedSymbol {
    readonly inline: boolean;
    readonly attributes: string[];
}
export declare class NamespaceSymbol extends ScopedSymbol implements INamespaceSymbol {
    readonly inline: boolean;
    readonly attributes: string[];
    constructor(name: string, inline?: boolean, attributes?: string[]);
}
