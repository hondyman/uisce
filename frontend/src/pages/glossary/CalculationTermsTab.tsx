import React, { useMemo, useState, useEffect } from 'react';
import { Box, IconButton, Tooltip, Chip, Button } from '@mui/material';
import { Add as AddIcon, ArrowBack as ArrowBackIcon } from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';
import { useAllSemanticData, CatalogNode } from '../../api/glossary';
import { useTranslation } from 'react-i18next';
import { devDebug } from '../../utils/devLogger';
import './BusinessTermsTab.css';

interface Props {
  searchTerm?: string;
  onCreateTerm?: () => void;
  onEditTerm?: (term: CatalogNode) => void;
  onDeleteTerm?: (term: CatalogNode) => void;
}

export const CalculationTermsTab: React.FC<Props> = ({ searchTerm, onCreateTerm, onEditTerm, onDeleteTerm }) => {
  const { t } = useTranslation();
  const { tenant, datasource } = useTenant();
  const { data, error } = useAllSemanticData();

  const [selectedTerm, setSelectedTerm] = useState<CatalogNode | null>(null);

  // Filter and search
  const calculationTerms = useMemo(() => {
    const src = (data as any)?.calculation_terms || [];
    if (!Array.isArray(src)) return [];
    const term = (searchTerm || '').toLowerCase();
    return src.filter((n: any) =>
      (n.node_name || '').toLowerCase().includes(term) ||
      (n.description || '').toLowerCase().includes(term)
    );
  }, [data, searchTerm]);

  useEffect(() => {
    devDebug('[CalculationTermsTab] datasource:', datasource?.id, 'tenant:', tenant?.id);
    devDebug('[CalculationTermsTab] terms count:', calculationTerms.length);
  }, [datasource?.id, tenant?.id, calculationTerms]);

  if (error) {
    return (
      <div className="business-terms-error">
        <h2>{t('error.loading', 'Error Loading')}</h2>
        <p>{error instanceof Error ? error.message : String(error)}</p>
      </div>
    );
  }

  const handleBack = () => setSelectedTerm(null);

  return (
    <div className="business-terms-tab-container">
      {/* Header */}
      <div className="business-terms-header">
        <h3>{t('tab.calculation_terms', 'Calculated Values')}</h3>
        <div className="header-actions">
          <Tooltip title={t('action.add_term', 'Add Term') as string}>
            <span>
              <IconButton color="primary" onClick={() => onCreateTerm && onCreateTerm()}>
                <AddIcon />
              </IconButton>
            </span>
          </Tooltip>
        </div>
      </div>

      {/* List or Details */}
      {!selectedTerm ? (
        <div className="business-terms-container">
          <div className="business-terms-list">
            {calculationTerms.length === 0 ? (
              <div className="business-terms-empty">
                <p>{t('empty.no_terms', 'No terms found')}</p>
              </div>
            ) : (
              <ul className="business-terms-items">
                {calculationTerms.map((term: any) => (
                  <li key={term.id} className={`business-term-item`}>
                    <div className="business-term-main" onClick={() => setSelectedTerm(term)}>
                      <div className="business-term-title-row">
                        <span className="business-term-name">{term.node_name || 'Untitled'}</span>
                        <Chip label={t('type.calculation', 'Calculation')} size="small" color="primary" />
                      </div>
                      {term.description && (
                        <div className="business-term-desc">{term.description}</div>
                      )}
                    </div>
                    <div className="business-term-actions">
                      <Button size="small" onClick={() => onEditTerm && onEditTerm(term)}>
                        {t('action.edit', 'Edit')}
                      </Button>
                      <Button size="small" color="error" onClick={() => onDeleteTerm && onDeleteTerm(term)}>
                        {t('action.delete', 'Delete')}
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ) : (
        <div className="business-term-details">
          <div className="details-header">
            <IconButton onClick={handleBack}>
              <ArrowBackIcon />
            </IconButton>
            <h4>{selectedTerm.node_name || t('label.details', 'Details')}</h4>
          </div>
          <Box sx={{ p: 2 }}>
            <p><strong>{t('label.name', 'Name')}:</strong> {selectedTerm.node_name}</p>
            <p><strong>{t('label.description', 'Description')}:</strong> {selectedTerm.description || t('label.none', 'None')}</p>
            <p><strong>{t('label.path', 'Qualified Path')}:</strong> {selectedTerm.qualified_path || '-'}</p>
            <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
              <Button variant="outlined" onClick={() => onEditTerm && onEditTerm(selectedTerm)}>{t('action.edit', 'Edit')}</Button>
              <Button variant="outlined" color="error" onClick={() => onDeleteTerm && onDeleteTerm(selectedTerm)}>{t('action.delete', 'Delete')}</Button>
            </Box>
          </Box>
        </div>
      )}
    </div>
  );
};

export default CalculationTermsTab;
