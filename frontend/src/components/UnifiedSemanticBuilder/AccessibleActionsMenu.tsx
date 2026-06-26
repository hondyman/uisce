import React, { useEffect, useRef } from 'react';
import ReactDOM from 'react-dom';
import * as TablerIcons from '@tabler/icons-react';
import IconArchive from '@tabler/icons-react/dist/esm/icons/IconArchive.mjs';
import IconBolt from '@tabler/icons-react/dist/esm/icons/IconBolt.mjs';
import IconAdjustments from '@tabler/icons-react/dist/esm/icons/IconAdjustments.mjs';
import type { ModelCatalogNode } from '../../types/model';

interface AccessibleActionsMenuProps {
  model: ModelCatalogNode;
  isOpen: boolean;
  onClose: () => void;
  anchorEl?: HTMLElement | null;
  onCreateCustom?: (baseModelKey: string) => void;
  onClone?: (baseModelKey: string) => void;
  onArchive?: (modelId: string, isCore: boolean, modelKey?: string) => void;
  onPublish?: (modelId: string) => void;
  onDraft?: (modelId: string) => void;
  onEdit?: (model: ModelCatalogNode) => void;
  onRename?: (model: ModelCatalogNode) => void;
  onGenerateModel?: (model: ModelCatalogNode) => void;
  // kept for backward-compat in tests/stories
  onInfo?: () => void;
}

const AccessibleActionsMenu: React.FC<AccessibleActionsMenuProps> = ({
  model,
  isOpen,
  onClose,
  anchorEl,
  onCreateCustom,
  onClone,
  onArchive,
  onPublish,
  onDraft,
  onEdit,
  onRename
  , onGenerateModel
}) => {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [stateMenuOpen, setStateMenuOpen] = React.useState(false);

  // Manage focus and dismissal
  useEffect(() => {
    const node = contentRef.current;
    if (!isOpen || !node) return;
    const items = Array.from(node.querySelectorAll<HTMLButtonElement>('.dropdown-item'));
    if (items.length) setTimeout(() => items[0].focus(), 0);

    const handleClickOutside = (e: MouseEvent) => {
      if (!node) return;
  // Close when clicking outside the menu (and outside anchor if provided)
  if (!node.contains(e.target as Node) && (!anchorEl || !anchorEl.contains(e.target as Node))) {
        onClose();
      }
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
      else if (e.key === 'Tab') {
        e.preventDefault();
        const items = Array.from(node.querySelectorAll<HTMLButtonElement>('.dropdown-item'));
        if (!items.length) return;
        if (e.shiftKey) items[items.length - 1].focus(); else items[0].focus();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleKey);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleKey);
    };
  }, [isOpen, onClose, anchorEl]);

  // Position menu relative to anchor; handles viewport flip (always run hook for stable order)
  useEffect(() => {
    const node = contentRef.current;
    if (!node) return; // keep hook order consistent
    if (!isOpen) { node.style.visibility = 'hidden'; return; }
    if (!anchorEl) { // no anchor, keep default positioning visible
      node.style.position = 'fixed';
      node.style.top = '80px';
      node.style.left = '80px';
      node.style.zIndex = '4000';
      node.style.visibility = 'visible';
      return;
    }
    const rect = anchorEl.getBoundingClientRect();
    const estimatedWidth = 180;
    const scrollY = window.scrollY;
    const scrollX = window.scrollX;
    let left = rect.right + scrollX - estimatedWidth;
    if (left < 8) left = 8;
    let top = rect.bottom + scrollY + 4;
    node.style.position = 'absolute';
    node.style.visibility = 'hidden';
    node.style.top = '0px';
    node.style.left = '-9999px';
    requestAnimationFrame(() => {
      if (!contentRef.current) return;
      const height = contentRef.current.offsetHeight || 0;
      const viewportBottom = scrollY + window.innerHeight;
      if (top + height + 8 > viewportBottom) {
        top = rect.top + scrollY - height - 4;
        if (top < scrollY + 4) top = rect.top + scrollY;
      }
      contentRef.current.style.top = `${top}px`;
      contentRef.current.style.left = `${left}px`;
      contentRef.current.style.zIndex = '4000';
      contentRef.current.style.visibility = 'visible';
    });
  }, [isOpen, anchorEl]);

  const shouldRender = isOpen; // render even without anchor (tests mount without one)

  if (!shouldRender) return null;

  const menu = (
    <div
      ref={contentRef}
      className="actions-dropdown floating positioned"
      aria-label="Model actions menu"
      onClick={(e) => e.stopPropagation()}
    >
          {/* Core/common actions: always include Create Custom / Clone / Delete to satisfy tests */}
          {model.metadata?.can_create && (
            <button type="button" className="dropdown-item btn-add-custom" onClick={() => { onCreateCustom?.(model.model_key); onClose(); }}>
              <TablerIcons.IconPlus size={14} />
              <span>Create Custom</span>
            </button>
          )}
          {/* Generate model action: only show if a handler was provided */}
          {onGenerateModel && (
            <button
              type="button"
              className="dropdown-item btn-generate-model"
              onClick={() => { onGenerateModel?.(model); onClose(); }}
            >
              <IconBolt size={14} />
              <span>Generate model</span>
            </button>
          )}
          <button type="button" className="dropdown-item btn-clone" onClick={() => { onClone?.(model.model_key); onClose(); }}>
            <TablerIcons.IconCopy size={14} />
            <span>Clone</span>
          </button>
          {/* Change state submenu */}
          <div className="dropdown-item has-submenu" role="group" aria-label="Change state">
            <button
              type="button"
              className="dropdown-item"
              onClick={() => setStateMenuOpen((v) => !v)}
              aria-haspopup="menu"
              aria-expanded={stateMenuOpen ? 'true' : undefined}
              title="Change model state"
            >
                <IconAdjustments size={14} />
              <span>Change state</span>
              <span aria-hidden="true" className={`submenu-caret ${stateMenuOpen ? 'open' : ''}`}></span>
            </button>
            {stateMenuOpen && (
              <div className="submenu" role="group" aria-label="Change state options">
                {/* Build candidate states excluding current, and respect archive gating */}
                {(['published','draft','archived'] as const)
                  .filter((s) => s !== (model.status as any))
                  .filter((s) => s !== 'archived' || ((model.is_core && !model.is_custom) || (model.is_custom && model.custom_model_exists)))
                  .map((s) => (
                    <button
                      key={s}
                      type="button"
                      className={`dropdown-item state-${s}`}
                      onClick={() => {
                        if (s === 'published') onPublish?.(model.id);
                        else if (s === 'draft') onDraft?.(model.id);
                        else if (s === 'archived') onArchive?.(model.id, !!model.is_core, model.model_key);
                        onClose();
                      }}
                    >
                      {s === 'published' && <TablerIcons.IconCheck size={14} />}
                      {s === 'draft' && <TablerIcons.IconAlertTriangle size={14} />}
                      {s === 'archived' && <IconArchive size={14} />}
                      <span className="submenu-label capitalize">{s}</span>
                    </button>
                  ))}
              </div>
            )}
          </div>

          {/* Custom model extra actions */}
          {model.is_custom && model.custom_model_exists && (
            <>
              <button type="button" className="dropdown-item btn-edit" onClick={() => { onEdit?.(model); onClose(); }}>
                <TablerIcons.IconEdit size={14} />
                <span>Edit</span>
              </button>
              <button type="button" className="dropdown-item btn-rename" onClick={() => { onRename?.(model); onClose(); }}>
                <TablerIcons.IconEdit size={14} />
                <span>Rename</span>
              </button>
            </>
          )}
        </div>
  );

  return ReactDOM.createPortal(menu, document.body);
};

export default AccessibleActionsMenu;
