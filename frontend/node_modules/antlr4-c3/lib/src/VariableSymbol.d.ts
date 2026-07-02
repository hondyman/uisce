import { TypedSymbol } from "./TypedSymbol.js";
import { IType } from "./types.js";
export declare class VariableSymbol extends TypedSymbol {
    value: unknown;
    constructor(name: string, value: unknown, type?: IType);
}
