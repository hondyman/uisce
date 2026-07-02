import { MethodSymbol } from "./MethodSymbol.js";
import { VariableSymbol } from "./VariableSymbol.js";
/** A field which belongs to a class or other outer container structure. */
export declare class FieldSymbol extends VariableSymbol {
    setter?: MethodSymbol;
    getter?: MethodSymbol;
}
