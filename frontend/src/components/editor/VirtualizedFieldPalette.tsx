// components/editor/VirtualizedFieldPalette.tsx
import React, { useMemo } from 'react';
import { AutoSizer, List } from 'react-virtualized';
import 'react-virtualized/styles.css';
import styles from './VirtualizedFieldPalette.module.css';

export interface VirtualField {
  id: string;
  label: string;
  type: string;
  required?: boolean;
}

export interface VirtualizedFieldPaletteProps<T = VirtualField> {
  fields: T[];
  renderItem: (field: T, index: number) => React.ReactNode;
  height?: number;
  onScroll?: (scrollOffset: number) => void;
}

interface RowRendererParams {
  index: number;
  key: string;
  style: React.CSSProperties;
}

interface AutoSizerParams {
  width: number;
  height: number;
}

interface ScrollParams {
  scrollTop: number;
}

/**
 * VirtualizedFieldPalette: Drop-in replacement for long field lists.
 * Maintains 60fps performance with hundreds of fields by rendering only visible rows.
 * 
 * Usage:
 * ```tsx
 * <VirtualizedFieldPalette
 *   fields={allFields}
 *   height={400}
 *   renderItem={(field) => (
 *     <div onClick={() => addField(field.id)}>{field.label}</div>
 *   )}
 * />
 * ```
 */
export const VirtualizedFieldPalette = React.forwardRef<any, VirtualizedFieldPaletteProps<any>>(
  ({ fields, renderItem, height = 320, onScroll }, ref) => {
    const rowHeight = useMemo(() => 64, []);

    const rowRenderer = ({ index, key, style }: RowRendererParams) => (
      <div key={key} style={style} className={styles.row}>
        {renderItem(fields[index], index)}
      </div>
    );

    return (
      <div className={styles.container} style={{ height }}>
        <AutoSizer>
          {({ width, height: autoHeight }: AutoSizerParams) => (
            <List
              ref={ref}
              width={width}
              height={autoHeight}
              rowHeight={rowHeight}
              rowCount={fields.length}
              rowRenderer={rowRenderer}
              overscanRowCount={6}
              onScroll={({ scrollTop }: ScrollParams) => onScroll?.(scrollTop)}
            />
          )}
        </AutoSizer>
      </div>
    );
  }
);

VirtualizedFieldPalette.displayName = 'VirtualizedFieldPalette';

export default VirtualizedFieldPalette;
