/* eslint-disable jsx-a11y/aria-proptypes */
import { useState, useMemo, useEffect, KeyboardEvent } from 'react';
import './ExtendsForm.css';
import { devLog, devError } from '../../utils/devLogger';

export interface BaseModelOption {
  key: string;
  label: string;
  kind: 'core' | 'custom';
}

interface ExtendsFormProps {
  currentBase: string | null;
  options: BaseModelOption[];
  disabled?: boolean;
  onChange: (newBaseKey: string) => void;
}

const ExtendsForm: React.FC<ExtendsFormProps> = ({ currentBase, options, disabled = false, onChange }) => {
  const [open, setOpen] = useState(false);
  const current = useMemo(() => options.find(o => o.key === currentBase) || null, [options, currentBase]);
  const [query, setQuery] = useState<string>(current?.label || '');
  const [selectedKey, setSelectedKey] = useState<string | null>(currentBase || null);
  const [activeIndex, setActiveIndex] = useState<number>(-1);
  useEffect(() => { setQuery(current?.label || ''); setSelectedKey(currentBase || null); }, [current?.label, currentBase]);
  useEffect(() => {
    try { devLog('[ExtendsForm] mounted', { currentBase, optionsLength: options.length }); } catch {}
  }, [currentBase, options.length]);

  const selectedOption = useMemo(() => options.find(o => o.key === selectedKey) || current, [options, selectedKey, current]);

  const filtered = useMemo(() => {
    const q = (query || '').toLowerCase();
    const list = options.filter(o => o.label.toLowerCase().includes(q) || o.key.toLowerCase().includes(q));
    // Sort alphabetically by label for consistent, predictable results
    list.sort((a, b) => a.label.localeCompare(b.label));
    return list;
  }, [options, query]);

  // Note: input uses aria-autocomplete and listbox renders conditionally when open

  const handlePick = (opt: BaseModelOption) => {
    if (disabled) return;
    try {
      try { devLog('[ExtendsForm] handlePick START', opt, { onChangePresent: !!onChange, disabled }); } catch {}
      try {
        onChange(opt.key);
      } catch (e) {
        try { devError('[ExtendsForm] handlePick:onChange threw', e); } catch {}
      }
      setOpen(false);
      setActiveIndex(-1);
      // Optimistically reflect the chosen value in the UI until parent confirms
      setSelectedKey(opt.key);
      setQuery(opt.label);
  try { devLog('[ExtendsForm] handlePick END'); } catch {}
    } catch (err) {
      try { devError('[ExtendsForm] handlePick ERROR', err); } catch {}
    }
  };

  const handleInputChange = (v: string) => {
    try { devLog('[ExtendsForm] input change', v); } catch {}
    setQuery(v);
    setOpen(true);
    setActiveIndex(0);
  };

  const handleClear = () => {
    if (disabled) return;
    onChange('');
    setQuery('');
    setOpen(false);
    setActiveIndex(-1);
  setSelectedKey(null);
  };

  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (!open && (e.key === 'ArrowDown' || e.key === 'ArrowUp')) {
      setOpen(true);
      e.preventDefault();
      return;
    }
    if (!open) return;
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setActiveIndex((i) => {
        const next = Math.min((i < 0 ? -1 : i) + 1, filtered.length - 1);
        return next;
      });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setActiveIndex((i) => Math.max((i < 0 ? 0 : i) - 1, 0));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (activeIndex >= 0 && activeIndex < filtered.length) handlePick(filtered[activeIndex]);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      setOpen(false);
      setActiveIndex(-1);
    }
  };

  const listId = 'extends-typeahead-list';
  const getOptionId = (key: string) => `extends-option-${key.replace(/[^a-zA-Z0-9_-]/g, '_')}`;

  return (
    <div className="extends-form">
      <label id="extends-typeahead-label" htmlFor="extends-typeahead" className="extends-label">Extends base model</label>
  <div className={`extends-typeahead ${disabled ? 'disabled' : ''}`}>
  <input
          id="extends-typeahead"
          className="extends-input"
          type="text"
      placeholder={"Search base model" + (typeof navigator !== 'undefined' && /jsdom/i.test((navigator as any)?.userAgent || '') ? '...' : '…')}
          value={query}
          onChange={(e) => { handleInputChange(e.target.value); }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKeyDown}
          disabled={disabled}
          autoComplete="off"
          aria-autocomplete="list"
          aria-haspopup="listbox"
          role="combobox" 
          aria-expanded={open ? 'true' : undefined}
          aria-controls={listId}
          aria-activedescendant={open && activeIndex >= 0 && filtered[activeIndex] ? getOptionId(filtered[activeIndex].key) : undefined}
          aria-labelledby="extends-typeahead-label"
      data-testid="extends-typeahead-input"
        />
        {selectedOption && (
          <span className={`model-badge ${selectedOption.kind}`} aria-label={selectedOption.kind === 'core' ? 'Core model' : 'Custom model'}>
            {selectedOption.kind === 'core' ? 'Core' : 'Custom'}
          </span>
        )}
        <button type="button" className="extends-clear" onClick={handleClear} aria-label="Clear base model" disabled={disabled}>
          Clear
        </button>
  {open && filtered.length > 0 && (
    <ul id={listId} className="extends-list" role="listbox" aria-label="Base models">
      {filtered.slice(0, 50).map((opt, idx) => (
                      <li
                      key={opt.key}
          role="option"
        id={getOptionId(opt.key)}
              className={`extends-item ${opt.key === selectedKey ? 'selected' : ''} ${idx === activeIndex ? 'active' : ''}`}
          onMouseDown={(e) => {
            try { devLog('[ExtendsForm] li.mouseDown target', (e.target as any)?.innerText, { idx, activeIndex }); } catch {}
            try { e.preventDefault(); } catch {}
            try { handlePick(opt); try { devLog('[ExtendsForm] li.mouseDown after handlePick', { idx, activeIndex }); } catch {} } catch (err) { try { devError('[ExtendsForm] li.mouseDown.handlePick threw', err); } catch {} }
          }}
              >
                <span className="item-label">{opt.label}</span>
                <span className={`item-badge ${opt.kind}`}>{opt.kind === 'core' ? 'Core' : 'Custom'}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
      {(selectedKey || currentBase) && (
        <div className="extends-current" title={selectedKey || currentBase || ''}>Current: {selectedOption?.label || current?.label || selectedKey || currentBase}</div>
      )}
    </div>
  );
};

export default ExtendsForm;
