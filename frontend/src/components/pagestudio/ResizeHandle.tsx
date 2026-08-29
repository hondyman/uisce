import React, { useCallback } from 'react';
import { ResizableBox } from 'react-resizable';
import 'react-resizable/css/styles.css';

interface ResizeHandleProps {
  width: number;
  onResize: (newWidth: number) => void;
  minWidth?: number;
  maxWidth?: number;
  children: React.ReactNode;
  containerSx?: object;
}

const TOTAL_COLUMNS = 12;
const MIN_COLUMN_SPAN = 1;
const MAX_COLUMN_SPAN = 12;

export const ResizeHandle: React.FC<ResizeHandleProps> = ({
  width,
  onResize,
  minWidth = 60,
  maxWidth = 800,
  children,
  containerSx,
}) => {
  const handleResize = useCallback(
    (_event: React.SyntheticEvent, { size }: { size: { width: number } }) => {
      const clamped = Math.max(minWidth, Math.min(maxWidth, size.width));
      onResize(clamped);
    },
    [minWidth, maxWidth, onResize]
  );

  return (
    <ResizableBox
      width={width}
      height={0}
      axis="x"
      resizeHandles={['e']}
      onResize={handleResize}
      minConstraints={[minWidth, 0]}
      maxConstraints={[maxWidth, 0]}
      dragConstraints={{ left: 0, right: 0 }}
      enable={{ right: true, left: false, top: false, bottom: false }}
    >
      <div style={{ position: 'relative', height: '100%', ...containerSx }}>{children}</div>
    </ResizableBox>
  );
};

export function columnSpanToWidth(columnSpan: number, containerWidth: number): number {
  return Math.round((columnSpan / TOTAL_COLUMNS) * containerWidth);
}

export function widthToColumnSpan(width: number, containerWidth: number): number {
  return Math.round((width / containerWidth) * TOTAL_COLUMNS);
}

export function clampColumnSpan(span: number): number {
  return Math.max(MIN_COLUMN_SPAN, Math.min(MAX_COLUMN_SPAN, span));
}

export default ResizeHandle;
