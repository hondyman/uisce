import { IType, ReferenceKind, TypeKind } from "./types.js";
import { ClassSymbol } from "./ClassSymbol.js";
import { FieldSymbol } from "./FieldSymbol.js";
import { MethodSymbol } from "./MethodSymbol.js";
import { ScopedSymbol } from "./ScopedSymbol.js";
export declare class InterfaceSymbol extends ScopedSymbol implements IType {
    reference: ReferenceKind;
    /** Typescript allows an interface to extend a class, not only interfaces. */
    readonly extends: Array<ClassSymbol | InterfaceSymbol>;
    constructor(name: string, ext: Array<ClassSymbol | InterfaceSymbol>);
    get baseTypes(): IType[];
    get kind(): TypeKind;
    /**
     * @param _includeInherited not used
     *
     * @returns a list of all methods.
     */
    getMethods(_includeInherited?: boolean): Promise<MethodSymbol[]>;
    /**
     * @param _includeInherited Not used.
     *
     * @returns all fields.
     */
    getFields(_includeInherited?: boolean): Promise<FieldSymbol[]>;
}
