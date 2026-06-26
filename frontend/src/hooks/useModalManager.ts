import { useState } from 'react';

export function useModalManager() {
    const [modals, setModals] = useState<Record<string, any>>({});

    const openModal = (id: string, props?: any) =>
        setModals((m) => ({ ...m, [id]: { open: true, props } }));

    const closeModal = (id: string) =>
        setModals((m) => ({ ...m, [id]: { ...(m[id] || {}), open: false } }));

    return { modals, openModal, closeModal };
}
