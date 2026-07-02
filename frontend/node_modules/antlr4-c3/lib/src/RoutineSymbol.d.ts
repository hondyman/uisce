import { ParameterSymbol } from "./ParameterSymbol.js";
import { ScopedSymbol } from "./ScopedSymbol.js";
import { VariableSymbol } from "./VariableSymbol.js";
import { IType } from "./types.js";
/** A standalone function/procedure/rule. */
export declare class RoutineSymbol extends ScopedSymbol {
    returnType?: IType;
    constructor(name: string, returnType?: IType);
    getVariables(_localOnly?: boolean): Promise<VariableSymbol[]>;
    getParameters(_localOnly?: boolean): Promise<ParameterSymbol[]>;
}
