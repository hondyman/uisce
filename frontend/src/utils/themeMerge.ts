// src/utils/themeMerge.ts
import { ThemeDefinition, TenantThemeOverride } from "../types/pageStudio";

export function mergeTheme(
    core: ThemeDefinition,
    override?: TenantThemeOverride
): ThemeDefinition {
    if (!override) return core;

    return {
        ...core,
        tokens: {
            colors: { ...core.tokens.colors, ...(override.overrides.colors || {}) },
            typography: { ...core.tokens.typography, ...(override.overrides.typography || {}) },
            spacing: { ...core.tokens.spacing, ...(override.overrides.spacing || {}) },
            borderRadius: override.overrides.borderRadius ?? core.tokens.borderRadius,
        },
    };
}
