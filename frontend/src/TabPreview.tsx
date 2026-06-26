interface TabPreviewProps {
  tabs: string[];
}

export default function TabPreview({ tabs }: TabPreviewProps) {
  return (
    <div className="tab-preview">
      {tabs.map(t => <span key={t} className="tab-chip">{t}</span>)}
    </div>
  );
}