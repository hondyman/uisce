# Menu Item Mapping - Old Structure to New Structure

## Migration Map

This document shows where each menu item moved from the old flat structure to the new categorized structure.

### CATALOG SECTION

#### Discovery & Exploration
| Old Location | New Location | Item |
|---|---|---|
| Fabric | Catalog → Discovery & Exploration | API Catalog |
| Fabric | Catalog → Discovery & Exploration | Schema Explorer |
| Fabric | Catalog → Discovery & Exploration | Views Catalog |

#### Glossary & Metadata
| Old Location | New Location | Item |
|---|---|---|
| Config | Catalog → Glossary & Metadata | Business Glossary |
| Config | Catalog → Glossary & Metadata | Catalog Setup |
| Config | Catalog → Glossary & Metadata | Data Domains |

---

### WEAVE SECTION

#### Bundles & Models
| Old Location | New Location | Item |
|---|---|---|
| Fabric | Weave → Bundles & Models | Bundles |
| Fabric | Weave → Bundles & Models | Model Generator |
| Fabric | Weave → Bundles & Models | Model Builder |
| Fabric | Weave → Bundles & Models | Calculations Library |

#### Semantic & Lineage
| Old Location | New Location | Item |
|---|---|---|
| Config | Weave → Semantic & Lineage | Semantic Mapper |
| Fabric | Weave → Semantic & Lineage | Claim Aware Lineage |
| Analytics | Weave → Semantic & Lineage | Drift Reports |

#### Governance & Policies
| Old Location | New Location | Item |
|---|---|---|
| Governance | Weave → Governance & Policies | Policy Management |
| Governance | Weave → Governance & Policies | Role Management |
| Governance | Weave → Governance & Policies | Access Intelligence |
| Governance | Weave → Governance & Policies | Access Debugger |

---

### ENTITY SECTION

#### Entity Management
| Old Location | New Location | Item |
|---|---|---|
| Config | Entity → Entity Management | Entity Manager |
| Config | Entity → Entity Management | Related Objects |
| (New) | Entity → Entity Management | Entity Config |

#### Business Processes
| Old Location | New Location | Item |
|---|---|---|
| Config | Entity → Business Processes | BP Builder |
| Fabric | Entity → Business Processes | BP Model Builder |
| (New) | Entity → Business Processes | Process Flows (Semantic Layout Builder) |

#### Administration
| Old Location | New Location | Item |
|---|---|---|
| Config | Entity → Administration | Tenants |
| Config | Entity → Administration | Validation Rules |
| Config | Entity → Administration | Dynamic UI Generator |
| Config | Entity → Administration | Query Builder |

#### Analytics & Monitoring
| Old Location | New Location | Item |
|---|---|---|
| Analytics | Entity → Analytics & Monitoring | Pre-aggregation Advisor |
| Analytics | Entity → Analytics & Monitoring | Frontier Explorer |
| Analytics | Entity → Analytics & Monitoring | Report Builder |
| Governance | Entity → Analytics & Monitoring | Notification Dashboard |

#### System & Upgrades
| Old Location | New Location | Item |
|---|---|---|
| Upgrade | Entity → System & Upgrades | Upgrade Center |
| Upgrade | Entity → System & Upgrades | Upgrade Compare |
| Governance | Entity → System & Upgrades | Notification Rules |
| Governance | Entity → System & Upgrades | Campaign Manager |

---

## Removed/Reorganized Items

### Removed from Top Navigation
None - all items have been preserved and reorganized into logical categories.

### Consolidated/Renamed Items
| Old Name | New Name | Location | Reason |
|---|---|---|---|
| N/A | Entity Config | Entity → Entity Management | Added for clarity (was implied under Entity Manager) |
| N/A | Process Flows | Entity → Business Processes | Renamed from "Semantic Layout Builder" for consistency |
| BP Builder (Fabric) | BP Model Builder | Entity → Business Processes | Moved from Fabric for consistency with Entity section |

---

## Item Count by Category

### Old Structure
- **Config:** 11 items
- **Fabric:** 9 items
- **Governance:** 7 items
- **Analytics:** 5 items
- **Upgrade:** 2 items
- **Total:** 34 items

### New Structure

#### Catalog: 6 items
- Discovery & Exploration: 3 items
- Glossary & Metadata: 3 items

#### Weave: 14 items
- Bundles & Models: 4 items
- Semantic & Lineage: 3 items
- Governance & Policies: 4 items
- (Quick Actions: 3 items)

#### Entity: 15 items
- Entity Management: 3 items
- Business Processes: 3 items
- Administration: 4 items
- Analytics & Monitoring: 4 items
- System & Upgrades: 4 items

**Total: 35 items** (1 new item added: Entity Config)

---

## Rationale for Organization

### Catalog Category
Groups all data discovery, browsing, and metadata management features. This is where users go to understand what data exists and how it's organized.

**Why these items together:**
- API Catalog - Browse available APIs
- Schema Explorer - Explore available data structures
- Views Catalog - See generated/resolved views
- Business Glossary - Understand semantic terms and relationships
- Catalog Setup - Configure the glossary structure
- Data Domains - Understand domain organization

### Weave Category
Groups all semantic fabric, bundling, policy, and lineage features. This is where users go to create and manage the semantic layer.

**Why these items together:**
- Bundles & Models - Create semantic packages and data models
- Semantic & Lineage - Map and trace data through the system
- Governance & Policies - Control access and enforce policies
- Quick Actions - Fast access to key Weave features

### Entity Category
Groups entity management, business processes, system administration, and analytics. This is where users go to manage the business model and system operations.

**Why these items together:**
- Entity Management - Define and manage entities
- Business Processes - Model and visualize processes
- Administration - Configure system and tenants
- Analytics & Monitoring - Monitor and optimize performance
- System & Upgrades - Manage infrastructure and upgrades

---

## Benefits of New Organization

1. **Clearer Mental Model**
   - Catalog = Discovery
   - Weave = Semantic Layer Management
   - Entity = Business Model & Operations

2. **Reduced Cognitive Load**
   - Fewer top-level categories (3 vs 5)
   - More subcategories for better organization
   - Logical groupings reduce search time

3. **Improved Workflow**
   - Users can anticipate where features are
   - Related features are near each other
   - Clear separation of concerns

4. **Scalability**
   - Easy to add new items to existing groups
   - New categories can be added without restructuring
   - Consistent structure as product grows

5. **Visual Communication**
   - Color coding helps users remember sections
   - Descriptions clarify purpose
   - Icons provide quick visual reference

---

## Implementation Notes

### Technical Changes
- Replaced `navigationGroups` with `navigationCategories`
- Each category includes color scheme (primary, light, dark, background)
- Groups now nested within categories
- Active state uses category color instead of primary theme color

### Styling Updates
- Category headers use category-specific background colors
- Item cards highlight in category color when selected
- Hover effects use category's light color
- Badges and accents respect category color scheme

### Backward Compatibility
- All routes remain unchanged
- No database migrations needed
- Frontend-only change to navigation
- Existing bookmarks and direct URLs still work

---

## Next Steps

1. Deploy changes to development/staging environment
2. User acceptance testing on each category
3. Gather feedback on subcategory organization
4. Monitor analytics for common navigation patterns
5. Iterate if needed based on usage patterns

---

## Questions & Support

- **Item seems in wrong place?** Review the "Rationale for Organization" section
- **Can't find a feature?** Check the "Item Mapping" table above
- **Want to reorganize items?** The structure is flexible - items can be moved between groups/categories
- **Need new items added?** Follow the pattern in `MainNavigation.tsx` navigationCategories array
