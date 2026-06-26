import React, { useState, useEffect } from 'react';
import { 
  Users, Building, FileText, ChevronRight, ChevronDown, 
  Plus, ArrowRight, DollarSign, Briefcase, Home
} from 'lucide-react';
import { fetchAPI } from '../../../api';

// Types
interface Entity {
  entity_id: string;
  entity_type: 'INDIVIDUAL' | 'JOINT' | 'TRUST' | 'LLC' | 'FOUNDATION' | 'ESTATE';
  entity_name: string;
  tax_id?: string;
  parent_entity_id?: string;
  household_id: string;
  // Specific fields
  trust_type?: string;
  foundation_type?: string;
  ownership_structure?: any;
}

interface HouseholdHierarchy {
  household_id: string;
  entities: Entity[];
}

interface HouseholdDashboardProps {
  householdId: string;
}

export const HouseholdDashboard: React.FC<HouseholdDashboardProps> = ({ householdId }) => {
  const [hierarchy, setHierarchy] = useState<HouseholdHierarchy | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedEntity, setSelectedEntity] = useState<Entity | null>(null);

  useEffect(() => {
    loadHierarchy();
  }, [householdId]);

  const loadHierarchy = async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<HouseholdHierarchy>(`/households/${householdId}/entities`);
      setHierarchy(data);
    } catch (error) {
      console.error('Failed to load household hierarchy:', error);
    } finally {
      setLoading(false);
    }
  };

  const buildTree = (entities: Entity[]) => {
    const map = new Map<string, any>();
    const roots: any[] = [];

    entities.forEach(e => {
      map.set(e.entity_id, { ...e, children: [] });
    });

    entities.forEach(e => {
      const node = map.get(e.entity_id);
      if (e.parent_entity_id && map.has(e.parent_entity_id)) {
        map.get(e.parent_entity_id).children.push(node);
      } else {
        roots.push(node);
      }
    });

    return roots;
  };

  if (loading) return <div className="p-8 text-center">Loading household structure...</div>;
  if (!hierarchy) return <div className="p-8 text-center">Household not found</div>;

  const tree = buildTree(hierarchy.entities);

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Household Structure</h1>
          <p className="text-gray-500 mt-1">Manage complex entity relationships and ownership</p>
        </div>
        <button className="px-4 py-2 bg-blue-600 text-white rounded-lg flex items-center gap-2 hover:bg-blue-700">
          <Plus size={16} /> Add Entity
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Entity Tree */}
        <div className="lg:col-span-1 bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="font-semibold text-lg mb-4">Entity Hierarchy</h2>
          <div className="space-y-2">
            {tree.map(node => (
              <EntityNode 
                key={node.entity_id} 
                node={node} 
                onSelect={setSelectedEntity} 
                selectedId={selectedEntity?.entity_id}
              />
            ))}
          </div>
        </div>

        {/* Entity Details */}
        <div className="lg:col-span-2">
          {selectedEntity ? (
            <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
              <div className="flex items-start justify-between mb-6">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-blue-50 text-blue-600 rounded-lg">
                    <EntityIcon type={selectedEntity.entity_type} />
                  </div>
                  <div>
                    <h2 className="text-xl font-bold text-gray-900">{selectedEntity.entity_name}</h2>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-medium px-2 py-0.5 bg-gray-100 text-gray-600 rounded">
                        {selectedEntity.entity_type}
                      </span>
                      {selectedEntity.tax_id && (
                        <span className="text-sm text-gray-500">Tax ID: {selectedEntity.tax_id}</span>
                      )}
                    </div>
                  </div>
                </div>
                <button className="text-sm text-blue-600 hover:underline">Edit Details</button>
              </div>

              <div className="grid grid-cols-2 gap-6 mb-8">
                <div className="p-4 bg-gray-50 rounded-lg">
                  <div className="text-sm text-gray-500 mb-1">Total Assets</div>
                  <div className="text-2xl font-bold text-gray-900">$24.5M</div>
                </div>
                <div className="p-4 bg-gray-50 rounded-lg">
                  <div className="text-sm text-gray-500 mb-1">YTD Performance</div>
                  <div className="text-2xl font-bold text-green-600">+8.4%</div>
                </div>
              </div>

              {selectedEntity.entity_type === 'TRUST' && (
                <div className="mb-6">
                  <h3 className="font-semibold mb-3">Trust Details</h3>
                  <div className="grid grid-cols-2 gap-4">
                    <DetailRow label="Trust Type" value={selectedEntity.trust_type || 'N/A'} />
                    <DetailRow label="Termination Date" value="2045-12-31" />
                    <DetailRow label="Trustees" value="Jane Doe, Bank of America" />
                    <DetailRow label="Beneficiaries" value="John Doe Jr., Sarah Doe" />
                  </div>
                </div>
              )}

              {selectedEntity.entity_type === 'LLC' && (
                <div className="mb-6">
                  <h3 className="font-semibold mb-3">Ownership Structure</h3>
                  <div className="bg-gray-50 p-4 rounded-lg">
                    {/* Mock ownership visualization */}
                    <div className="flex justify-between items-center mb-2">
                      <span>John Doe (Member)</span>
                      <span className="font-medium">50%</span>
                    </div>
                    <div className="w-full bg-gray-200 h-2 rounded-full overflow-hidden">
                      <div className="bg-blue-600 h-full w-1/2"></div>
                    </div>
                  </div>
                </div>
              )}

            </div>
          ) : (
            <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-12 text-center text-gray-500">
              <Users size={48} className="mx-auto mb-4 text-gray-300" />
              <p>Select an entity to view details</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const EntityNode: React.FC<{ node: any; onSelect: (e: Entity) => void; selectedId?: string; level?: number }> = ({ 
  node, onSelect, selectedId, level = 0 
}) => {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = node.children && node.children.length > 0;

  return (
    <div className="select-none">
      <div 
        className={`flex items-center gap-2 p-2 rounded-lg cursor-pointer transition-colors ${
          selectedId === node.entity_id ? 'bg-blue-50 text-blue-700' : 'hover:bg-gray-50'
        }`}
        style={{ paddingLeft: `${level * 1.5 + 0.5}rem` }}
        onClick={() => onSelect(node)}
      >
        <button 
          onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}
          className={`p-0.5 rounded hover:bg-gray-200 ${hasChildren ? 'visible' : 'invisible'}`}
        >
          {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </button>
        <EntityIcon type={node.entity_type} size={16} />
        <span className="text-sm font-medium truncate">{node.entity_name}</span>
      </div>
      
      {expanded && hasChildren && (
        <div>
          {node.children.map((child: any) => (
            <EntityNode 
              key={child.entity_id} 
              node={child} 
              onSelect={onSelect} 
              selectedId={selectedId} 
              level={level + 1} 
            />
          ))}
        </div>
      )}
    </div>
  );
};

const EntityIcon: React.FC<{ type: string; size?: number }> = ({ type, size = 20 }) => {
  switch (type) {
    case 'INDIVIDUAL': return <Users size={size} />;
    case 'JOINT': return <Users size={size} />;
    case 'TRUST': return <FileText size={size} />;
    case 'LLC': return <Briefcase size={size} />;
    case 'FOUNDATION': return <Building size={size} />;
    case 'ESTATE': return <Home size={size} />;
    default: return <Users size={size} />;
  }
};

const DetailRow: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div>
    <div className="text-xs text-gray-500 uppercase">{label}</div>
    <div className="text-sm font-medium text-gray-900">{value}</div>
  </div>
);

export default HouseholdDashboard;
