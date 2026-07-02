import { TypedSymbol } from "./TypedSymbol.js";
import { IType } from "./types.js";
export declare class LiteralSymbol extends TypedSymbol {
    readonly value: unknown;
    constructor(name: string, value: unknown, type?: IType);
}
