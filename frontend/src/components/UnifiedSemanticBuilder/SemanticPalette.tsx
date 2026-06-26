import React, { useState, useMemo } from 'react';
import { IconDatabase, IconFilter, IconChartBar, IconPlugConnected, IconStack3 } from '@tabler/icons-react';
import { GitBranch as LucideGitBranch, Info as LucideInfo } from 'lucide-react';
import PaletteItem from './PaletteItem';
import './SemanticPalette.css';
import './SemanticPaletteHorizontal.css';
import styles from './SemanticPalette.module.css';

interface SemanticPaletteProps {
  onAdd: (type: 'dimension' | 'measure' | 'filter' | 'join' | 'extends') => void;
  horizontal?: boolean;
  enableDrag?: boolean; // keep drag temporarily (optional)
  canAddExtends?: boolean; // when true, Extends action is enabled
  extendsDisabledReason?: string; // optional reason to show when Extends is disabled
}

interface SimpleElement {
  type: 'dimension' | 'measure' | 'filter' | 'join' | 'pre_aggregation';
  label: string;
  description: string;
  icon: React.ReactNode;
}

const ELEMENTS: SimpleElement[] = [
  { type: 'dimension', label: 'Dimension', description: 'Add a new dimension or override core', icon: <IconDatabase size={16} /> },
  { type: 'measure', label: 'Measure', description: 'Add a new measure or override core', icon: <IconChartBar size={16} /> },
  { type: 'filter', label: 'Filter', description: 'Add a new filter or override core', icon: <IconFilter size={16} /> },
  { type: 'join', label: 'Join', description: 'Define a join (future)', icon: <IconPlugConnected size={16} /> },
  { type: 'pre_aggregation', label: 'Pre-Aggregation', description: 'View configured pre-aggregations', icon: <IconStack3 size={16} /> }
];

const SemanticPalette: React.FC<SemanticPaletteProps> = ({ onAdd, horizontal = true, enableDrag = false, canAddExtends = false, extendsDisabledReason }) => {
  const [tooltip, setTooltip] = useState<{ show: boolean; x: number; y: number; label?: string; description?: string }>({ show: false, x: 0, y: 0, label: undefined, description: undefined });

  // Filter elements based on search term
  const filteredElements = useMemo(() => {
    return ELEMENTS;
  }, []);

  const handleTooltipShow = (label: string, description: string, rect: DOMRect) => {
    setTooltip({ show: true, x: rect.left + rect.width / 2, y: rect.top - 10, label, description });
  };
  const handleTooltipHide = () => setTooltip({ show: false, x: 0, y: 0 });

  return (
    <>
      <div className={`semantic-palette minimal ${horizontal ? 'horizontal' : ''}`}>
        <div className="palette-toolbar simple colored">
          <div className="toolbar-items simple colored">
            {/* Always render the Extends button but disable it when not allowed. If disabled, show a helpful tooltip. */}
            <div className="extends-button-wrapper">
            <button
              key="extends"
              className={`palette-icon-btn colored extends ${canAddExtends ? '' : 'disabled'}`}
              type="button"
              onClick={() => { if (canAddExtends) onAdd('extends'); }}
              onMouseEnter={(e) => {
                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                if (!canAddExtends && extendsDisabledReason) {
                  handleTooltipShow('Extends', extendsDisabledReason, rect);
                } else {
                  handleTooltipShow('Extends', 'Set the base model this model extends', rect);
                }
              }}
              onMouseLeave={() => handleTooltipHide()}
              aria-label="Extends"
              title={canAddExtends ? 'Extends' : (extendsDisabledReason || 'Extends')}
              disabled={!canAddExtends}
              aria-disabled={!canAddExtends}
            >
              <LucideGitBranch size={16} />
              {!horizontal && <span className="label">Extends</span>}
              {horizontal && <span className="sr-only">Extends</span>}
            </button>
            {/* Small inline visual indicator when the Extends action is disabled */}
            {!canAddExtends && extendsDisabledReason && (
              <button
                className="extends-disabled-indicator"
                type="button"
                aria-label={extendsDisabledReason}
                title={extendsDisabledReason}
                onMouseEnter={(e) => handleTooltipShow('Extends', extendsDisabledReason, (e.currentTarget as HTMLElement).getBoundingClientRect())}
                onMouseLeave={() => handleTooltipHide()}
                onClick={() => { /* noop - purely informational */ }}
              >
                <LucideInfo size={14} />
              </button>
            )}
            </div>
            {filteredElements.map(el => (
              el.type === 'pre_aggregation' ? (
                <button
                  key={el.type}
                  className={`palette-icon-btn colored pre_aggregation`}
                  type="button"
                  onMouseEnter={(e) => { handleTooltipShow(el.label, el.description, (e.currentTarget as HTMLElement).getBoundingClientRect()); }}
                  onMouseLeave={() => handleTooltipHide()}
                  aria-label="Pre-Aggregation"
                  title="Pre-Aggregation"
                >
                  {el.icon}
                  {!horizontal && <span className="label">{el.label}</span>}
                  {horizontal && <span className="sr-only">{el.label}</span>}
                </button>
              ) : (
                <PaletteItem
                  key={el.type}
                  typeName={el.type as any}
                  label={el.label}
                  description={el.description}
                  icon={el.icon}
                  onAdd={onAdd}
                  horizontal={horizontal}
                  enableDrag={enableDrag}
                  onTooltipShow={handleTooltipShow}
                  onTooltipHide={handleTooltipHide}
                />
              )
            ))}
          </div>
        </div>
      </div>
      {tooltip.show && (
        <div className={`palette-tooltip ${styles.paletteTooltip}`} data-left={tooltip.x} data-top={tooltip.y}>
          <div className={styles.tooltipTitle}>{tooltip.label}</div>
          {tooltip.description && <div className={styles.tooltipDesc}>{tooltip.description}</div>}
        </div>
      )}
    </>
  );
};

export default SemanticPalette;
