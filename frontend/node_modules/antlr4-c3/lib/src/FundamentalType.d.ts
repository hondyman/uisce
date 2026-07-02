import { IType, TypeKind, ReferenceKind } from "./types.js";
/** A single class for all fundamental types. They are distinguished via the kind field. */
export declare class FundamentalType implements IType {
    static readonly integerType: FundamentalType;
    static readonly floatType: FundamentalType;
    static readonly stringType: FundamentalType;
    static readonly boolType: FundamentalType;
    name: string;
    private typeKind;
    private referenceKind;
    constructor(name: string, typeKind?: TypeKind, referenceKind?: ReferenceKind);
    get baseTypes(): IType[];
    get kind(): TypeKind;
    get reference(): ReferenceKind;
}
