import { useState } from 'react';

export function usePageState(initial?: Record<string, any>) {
    const [state, setState] = useState<Record<string, any>>(initial || {});
    return { state, setState };
}
