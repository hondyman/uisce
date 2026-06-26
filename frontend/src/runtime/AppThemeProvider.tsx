// src/runtime/AppThemeProvider.tsx
import React, { useMemo } from 'react';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { ThemeDefinition, TenantThemeOverride } from '../types/pageStudio';
import { mergeTheme } from '../utils/themeMerge';

interface AppThemeProviderProps {
    coreTheme: ThemeDefinition;
    tenantOverride?: TenantThemeOverride;
    children: React.ReactNode;
}

export const AppThemeProvider: React.FC<AppThemeProviderProps> = ({ coreTheme, tenantOverride, children }) => {
    const theme = useMemo(() => {
        const merged = mergeTheme(coreTheme, tenantOverride);
        
        return createTheme({
            palette: {
                primary: {
                    main: merged.tokens.colors.primary || '#3b82f6',
                },
                secondary: {
                    main: merged.tokens.colors.secondary || '#6366f1',
                },
                background: {
                    default: merged.tokens.colors.background || '#f8fafc',
                }
            },
            shape: {
                borderRadius: merged.tokens.borderRadius || 8,
            },
            typography: {
                fontFamily: merged.tokens.typography.fontFamily || '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
                h4: {
                    fontWeight: 700,
                }
            },
            components: {
                MuiButton: {
                    styleOverrides: {
                        root: {
                            textTransform: 'none',
                            fontWeight: 600,
                        }
                    }
                },
                MuiPaper: {
                    styleOverrides: {
                        root: {
                            boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
                        }
                    }
                }
            }
        });
    }, [coreTheme, tenantOverride]);

    return (
        <ThemeProvider theme={theme}>
            {children}
        </ThemeProvider>
    );
};
