import { useEffect, useState } from "react";

export function usePersistedForm<T>(
    storageKey: string,
    defaultValue: T,
    scope: "local" | "session" = "local"
) {
    const storage = scope === "local" ? window.localStorage : window.sessionStorage;

    const [state, setState] = useState<T>(() => {
        try {
            const raw = storage.getItem(storageKey);
            if (!raw) return defaultValue;
            return JSON.parse(raw) as T;
        } catch {
            return defaultValue;
        }
    });

    useEffect(() => {
        storage.setItem(storageKey, JSON.stringify(state));
    }, [storageKey, state, storage]);

    // Clear handler for manual reset
    const clearState = () => {
        storage.removeItem(storageKey);
        setState(defaultValue);
    };

    return [state, setState, clearState] as const;
}
