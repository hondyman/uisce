import { useCallback, useState } from 'react';

export type FieldErrorMap = Record<string, string[]>;

export interface ValidationErrorDetail {
    field?: string;
    message?: string;
}

export interface ValidationErrorResponse {
    error?: string;
    message?: string;
    details?: ValidationErrorDetail[];
}

const DEFAULT_VALIDATION_MESSAGE = 'Validation failed. Please fix the highlighted fields.';

const mapFieldErrors = (details?: ValidationErrorDetail[] | null): FieldErrorMap => {
    if (!details || details.length === 0) {
        return {};
    }

    return details.reduce<FieldErrorMap>((acc, detail) => {
        if (!detail?.field) {
            return acc;
        }

        const key = detail.field;
        const message = detail.message || 'Invalid value';

        if (!acc[key]) {
            acc[key] = [];
        }
        acc[key].push(message);
        return acc;
    }, {} as FieldErrorMap);
};

const removeFieldKey = (map: FieldErrorMap, key: string) => {
    if (!map[key]) {
        return map;
    }

    const next = { ...map };
    delete next[key];
    return next;
};

export const useValidationErrors = () => {
    const [fieldErrors, setFieldErrors] = useState<FieldErrorMap>({});

    const clearFieldErrors = useCallback(() => {
        setFieldErrors({});
    }, []);

    const clearFieldError = useCallback((path: string) => {
        setFieldErrors((prev) => {
            if (!prev || Object.keys(prev).length === 0) {
                return prev;
            }

            let changed = false;
            let next = prev;

            if (prev[path]) {
                next = removeFieldKey(next, path);
                changed = true;
            }

            const prefixDot = `${path}.`;
            const prefixBracket = `${path}[`;

            Object.keys(prev).forEach((key) => {
                if (key.startsWith(prefixDot) || key.startsWith(prefixBracket)) {
                    if (!changed) {
                        next = { ...next };
                        changed = true;
                    }
                    delete next[key];
                }
            });

            return changed ? next : prev;
        });
    }, []);

    const applyValidationDetails = useCallback((details?: ValidationErrorDetail[] | null) => {
        setFieldErrors(mapFieldErrors(details));
    }, []);

    const getFieldErrors = useCallback(
        (path: string): string[] => fieldErrors[path] ?? [],
        [fieldErrors]
    );

    const hasFieldError = useCallback(
        (path: string): boolean => getFieldErrors(path).length > 0,
        [getFieldErrors]
    );

    const fieldHelperText = useCallback(
        (path: string): string | undefined => {
            const messages = getFieldErrors(path);
            return messages.length > 0 ? messages.join(' ') : undefined;
        },
        [getFieldErrors]
    );

    const handleResponseError = useCallback(
        async (response: Response, fallbackMessage = DEFAULT_VALIDATION_MESSAGE): Promise<never> => {
            const contentType = response.headers.get('content-type') || '';

            if (response.status === 422 && contentType.includes('application/json')) {
                let payload: ValidationErrorResponse | null = null;
                try {
                    payload = (await response.clone().json()) as ValidationErrorResponse;
                } catch {
                    payload = null;
                }

                if (payload) {
                    applyValidationDetails(payload.details);
                    const message = payload.message || fallbackMessage;
                    throw new Error(message);
                }
            }

            let message = '';
            try {
                message = await response.text();
            } catch {
                message = '';
            }
            throw new Error(message || `Request failed with status ${response.status}`);
        },
        [applyValidationDetails]
    );

    return {
        fieldErrors,
        clearFieldErrors,
        clearFieldError,
        applyValidationDetails,
        getFieldErrors,
        hasFieldError,
        fieldHelperText,
        handleResponseError
    };
};

export type UseValidationErrorsReturn = ReturnType<typeof useValidationErrors>;
