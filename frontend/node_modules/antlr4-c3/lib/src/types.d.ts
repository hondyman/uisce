/** Visibility (aka. accessibility) of a symbol member. */
export declare enum MemberVisibility {
    /** Not specified, default depends on the language and type. */
    Unknown = 0,
    /** Used in Swift, member can be accessed outside of the defining module and extended. */
    Open = 1,
    /** Like Open, but in Swift such a type cannot be extended. */
    Public = 2,
    /** Member is only accessible in the defining class and any derived class. */
    Protected = 3,
    /** Member can only be accessed from the defining class. */
    Private = 4,
    /**
     * Used in Swift and Java, member can be accessed from everywhere in a defining module, not outside however.
     * Also known as package private.
     */
    FilePrivate = 5,
    /** Custom enum for special usage. */
    Library = 6
}
/** The modifier of a symbol member. */
export declare enum Modifier {
    Static = 0,
    Final = 1,
    Sealed = 2,
    Abstract = 3,
    Deprecated = 4,
    Virtual = 5,
    Const = 6,
    Overwritten = 7
}
/** Rough categorization of a type. */
export declare enum TypeKind {
    Unknown = 0,
    Integer = 1,
    Float = 2,
    Number = 3,
    String = 4,
    Char = 5,
    Boolean = 6,
    Class = 7,
    Interface = 8,
    Array = 9,
    Map = 10,
    Enum = 11,
    Alias = 12
}
/** Describes a reference to a type. */
export declare enum ReferenceKind {
    Irrelevant = 0,
    /** Default for most languages for dynamically allocated memory ("Type*" in C++). */
    Pointer = 1,
    /** "Type&" in C++, all non-primitive types in Java/Javascript/Typescript etc. */
    Reference = 2,
    /** "Type" as such and default for all value types. */
    Instance = 3
}
/** The root type interface. Used for typed symbols and type aliases. */
export interface IType {
    name: string;
    /**
     * The super type of this type or empty if this is a fundamental type.
     * Also used as the target type for type aliases.
     */
    baseTypes: IType[];
    kind: TypeKind;
    reference: ReferenceKind;
}
export interface ISymbolTableOptions {
    allowDuplicateSymbols?: boolean;
}
