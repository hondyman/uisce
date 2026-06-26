import { useMemo } from 'react';
import { useDrag } from 'react-dnd';
import './Palette.css';
import * as Icons from './icons';
import styles from './Palette.module.css';

export type TileKind = 'dimension' | 'measure' | 'join' | 'filter' | 'blank';

interface TileSpec {
  kind: TileKind;
  title: string;
  color: string;
  icon: any;
}

const TILES: TileSpec[] = [
  { kind: 'dimension', title: 'Dimension', color: '#60a5fa', icon: Icons.IconTextSize },
  { kind: 'measure', title: 'Measure', color: '#34d399', icon: Icons.IconNumbers },
  { kind: 'join', title: 'Join', color: '#f59e0b', icon: Icons.IconStack3 },
  { kind: 'filter', title: 'Filter', color: '#f97316', icon: Icons.IconFilter },
];

export default function Palette({
  coreItems = [],
  customItems = [],
}: {
  coreItems?: any[];
  customItems?: any[];
}) {
  // Precomputed list of searchable items (reserved for future search UI).
  // Keep computation so consumers can opt-in to search without changing this file.
  const _allSearchableItems = useMemo(() => {
    const items = [
      ...TILES.map((tile) => ({
        id: `tile-${tile.kind}`,
        title: tile.title,
        description: `Core ${tile.kind} element`,
        meta: 'Core',
      })),
      ...coreItems.map((item, index) => ({
        id: `core-${item.id || item.name || index}`,
        title: item.name || item.id || `Core Item ${index + 1}`,
        description: item.description || 'Core database item',
        meta: 'Core Item',
      })),
      ...customItems.map((item, index) => ({
        id: `custom-${item.id || item.name || index}`,
        title: item.name || item.id || `Custom Item ${index + 1}`,
        description: item.description || 'Custom database item',
        meta: 'Custom Item',
      })),
    ];
    return items;
  }, [coreItems, customItems]);
  // reference so TS doesn't complain about unused computed value; consumers may use it later
  void _allSearchableItems;
  return (
    <div className="palette">
      <style>{`
        ${[...TILES.map(t => `.palette-tile-${t.kind} { --tile-color: ${t.color}; }`),
          ...Array.from({ length: Math.min(6, coreItems.length) }).map((_, i: number) => `.palette-core-item-${i} { --tile-color: #cbd5e0; }`),
          ...Array.from({ length: Math.min(6, customItems.length) }).map((_, i: number) => `.palette-custom-item-${i} { --tile-color: #60a5fa; }`)
        ].join('\n')}
      `}</style>
      <div className="palette-section">
        <div className="palette-section-title">Core Objects</div>
        <div className="palette-tiles">
          {TILES.map((t) => (
            <PaletteTile key={`core-${t.kind}`} spec={t} origin="core" generatedClass={`palette-tile-${t.kind}`} />
          ))}
          {coreItems.slice(0, 6).map((it: any) => (
            <PaletteTile
              key={`core-item-${it.id || it.name}`}
              spec={{ kind: 'blank' as TileKind, title: it.name || it.id || 'Core', color: '#cbd5e0', icon: Icons.IconDatabase }}
              origin="core"
              meta={it}
              generatedClass={`palette-core-item-${coreItems.indexOf(it)}`}
            />
          ))}
        </div>
      </div>

      <div className="palette-section">
        <div className="palette-section-title">Custom Objects</div>
        <div className="palette-tiles">
          {TILES.map((t) => (
            <PaletteTile key={`custom-${t.kind}`} spec={t} origin="custom" generatedClass={`palette-tile-${t.kind}`} />
          ))}
          <PaletteTile spec={{ kind: 'blank', title: 'New Object', color: '#60a5fa', icon: Icons.IconPlus }} origin="custom" />
          {customItems.slice(0, 6).map((it: any) => (
            <PaletteTile
              key={`custom-item-${it.id || it.name}`}
              spec={{ kind: 'blank' as TileKind, title: it.name || it.id || 'Custom', color: '#60a5fa', icon: Icons.IconDatabase }}
              origin="custom"
              meta={it}
              generatedClass={`palette-custom-item-${customItems.indexOf(it)}`}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function PaletteTile({ spec, origin, meta, generatedClass }: { spec: TileSpec; origin: 'core' | 'custom'; meta?: any; generatedClass?: string }) {
  const [{ isDragging }, drag] = useDrag(
    () => ({
      type: 'TILE',
      item: { kind: spec.kind, origin, meta },
      collect: (m: any) => ({ isDragging: !!m.isDragging() }),
    }),
    [spec, origin, meta]
  );

  const Icon = spec.icon as any;

  return (
    <div
      ref={drag}
      className={`palette-tile ${origin === 'core' ? 'core' : 'custom'} ${isDragging ? 'dragging' : ''} ${styles.paletteTile} ${generatedClass || ''}`}
      title={`${spec.title} (${origin})`}
    >
      <div className={`tile-icon ${styles.tileIcon}`}>
        <Icon size={16} />
      </div>
      <div className="tile-label">{spec.title}</div>
      {origin === 'core' && <div className="tile-readonly">Core</div>}
    </div>
  );
}
