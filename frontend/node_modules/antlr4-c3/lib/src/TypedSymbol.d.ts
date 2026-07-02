import { IType } from "./types.js";
import { BaseSymbol } from "./BaseSymbol.js";
/** A symbol with an attached type (variables, fields etc.). */
export declare class TypedSymbol extends BaseSymbol {
    type: IType | undefined;
    constructor(name: string, type?: IType);
}
