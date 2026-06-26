import { useState, useEffect } from 'react';
import { accessRulesApi } from '../../../api/accessRules';

interface ValidationResult {
    valid: boolean;
    error?: string;
    sql?: string;
    warnings?: string[];
}

export const useRuleValidation = (
    rowFilterDsl: string,
    businessObjectId: string,
    debounceMs: number = 500
) => {
    const [validation, setValidation] = useState<ValidationResult | null>(null);
    const [validating, setValidating] = useState(false);

    useEffect(() => {
        if (!rowFilterDsl || !businessObjectId) {
            setValidation(null);
            return;
        }

        setValidating(true);
        const timeoutId = setTimeout(async () => {
            try {
                const result = await accessRulesApi.validate({
                    rowFilterDsl,
                    businessObjectId,
                });
                setValidation(result);
            } catch (error: any) {
                setValidation({
                    valid: false,
                    error: error?.message || 'Validation failed',
                });
            } finally {
                setValidating(false);
            }
        }, debounceMs);

        return () => clearTimeout(timeoutId);
    }, [rowFilterDsl, businessObjectId, debounceMs]);

    return { validation, validating };
};

export default useRuleValidation;
