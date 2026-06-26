import type { FC } from 'react';
import './RelatedListConfigurator.css';

export const RelatedListConfigurator: FC<{
  section: any;
  primaryBO: any;
  onUpdate: (section: any) => void;
}> = ({ section, primaryBO, onUpdate }) => {
  const relationship = primaryBO.relationships.find((r: any) => r.id === section.relationshipId);

  return (
    <div>
      <div style={{ marginBottom: '1rem' }}>
        <select
          value={section.relationshipId || ''}
          onChange={(e) => {
            const rel = primaryBO.relationships.find((r: any) => r.id === e.target.value);
            onUpdate({ ...section, relationshipId: e.target.value, relatedBO: rel?.relatedBO });
          }}
          className="related-list-configurator__select"
          title="Select relationship"
        >
          <option value="">Select relationship…</option>
          {primaryBO.relationships.map((rel: any) => (
            <option key={rel.id} value={rel.id}>{rel.label} ({rel.relatedBO})</option>
          ))}
        </select>
      </div>

      {relationship && (
        <>
          <div style={{ marginBottom: '1rem', color: '#6b7280' }}>
            Related via <b>{relationship.foreignKey}</b> to <b>{relationship.relatedBO}</b>
          </div>
          {/* Example: choose visible columns */}
          <div style={{ marginBottom: '1rem' }}>
            <label style={{ display: 'block', fontWeight: 600, marginBottom: 8 }}>Columns</label>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8 }}>
              {(() => {
                const allFields = [...(primaryBO.coreFields || []), ...(primaryBO.customFields || primaryBO.fields || [])];
                return allFields.slice(0, 8).map((f: any) => (
                  <label key={f.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                    <input
                      type="checkbox"
                      checked={section.columnFieldIds?.includes(f.id) || false}
                      onChange={(e) => {
                        const next = new Set(section.columnFieldIds || []);
                        e.target.checked ? next.add(f.id) : next.delete(f.id);
                        onUpdate({ ...section, columnFieldIds: Array.from(next) });
                      }}
                    />
                    <span>{f.label}</span>
                  </label>
                ));
              })()}
            </div>
          </div>
        </>
      )}
    </div>
  );
};