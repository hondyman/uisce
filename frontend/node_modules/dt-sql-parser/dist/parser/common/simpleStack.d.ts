export declare class SimpleStack<T> {
    constructor();
    private stack;
    push(item: T): void;
    pop(): T;
    peek(): T;
    clear(): void;
    size(): number;
    isEmpty(): boolean;
}
