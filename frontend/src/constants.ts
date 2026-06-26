// Compact layout configuration for lineage visualization
export const LINEAGE_LAYOUT = {
  columnSpacing: 280,
  itemSpacing: 100,
  startX: 50,
  startY: 80,
  nodeWidth: 180,
  nodeHeight: 60,
  padding: 40
};

export const createLineagePosition = (column: number, row: number, totalInColumn: number) => {
  const centerY = LINEAGE_LAYOUT.startY + (LINEAGE_LAYOUT.itemSpacing * Math.max(totalInColumn - 1, 0)) / 2;
  const itemY = centerY - (LINEAGE_LAYOUT.itemSpacing * Math.max(totalInColumn - 1, 0)) / 2 + (row * LINEAGE_LAYOUT.itemSpacing);
  
  return {
    x: LINEAGE_LAYOUT.startX + (column * LINEAGE_LAYOUT.columnSpacing),
    y: itemY
  };
};
