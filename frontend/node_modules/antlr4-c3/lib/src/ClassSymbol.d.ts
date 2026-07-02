import { IType, ReferenceKind, TypeKind } from "./types.js";
import { FieldSymbol } from "./FieldSymbol.js";
import { InterfaceSymbol } from "./InterfaceSymbol.js";
import { MethodSymbol } from "./MethodSymbol.js";
import { ScopedSymbol } from "./ScopedSymbol.js";
/** Classes and structs. */
export declare class ClassSymbol extends ScopedSymbol implements IType {
    isStruct: boolean;
    reference: ReferenceKind;
    /** Usually only one member, unless the language supports multiple inheritance (like C++). */
    readonly extends: ClassSymbol[];
    /** Typescript allows a class to implement a class, not only interfaces. */
    readonly implements: Array<ClassSymbol | InterfaceSymbol>;
    constructor(name: string, ext: ClassSymbol[], impl: Array<ClassSymbol | InterfaceSymbol>);
    get baseTypes(): IType[];
    get kind(): TypeKind;
    /**
     * @param _includeInherited Not used.
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
