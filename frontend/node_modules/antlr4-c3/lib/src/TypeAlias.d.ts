import { ReferenceKind, IType, TypeKind } from "./types.js";
import { BaseSymbol } from "./BaseSymbol.js";
/** An alias for another type. */
export declare class TypeAlias extends BaseSymbol implements IType {
    private targetType;
    constructor(name: string, target: IType);
    get baseTypes(): IType[];
    get kind(): TypeKind;
    get reference(): ReferenceKind;
}
