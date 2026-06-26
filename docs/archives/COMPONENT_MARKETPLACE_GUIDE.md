<!-- cspell:disable -->
# Component Marketplace - Implementation Guide

## Overview

The Component Marketplace is a feature-rich discovery and installation platform for dashboard components integrated into the Weave menu. It enables users to browse, search, filter, and install pre-built components for their dashboards.

## Architecture

The marketplace is built with a modular, scalable architecture:

```
├── contexts/
│   └── MarketplaceContext.tsx          # Centralized state management
├── components/marketplace/
│   ├── index.ts                        # Main exports
│   ├── ComponentMarketplace.tsx        # Main component orchestrator
│   ├── SearchBar.tsx                   # Debounced search input
│   ├── CategoryFilter.tsx              # Category selector
│   ├── ComponentCard.tsx               # Grid card component
│   ├── ComponentModal.tsx              # Detail modal
│   └── FeaturedComponents.tsx          # Featured section
├── pages/marketplace/
│   └── ComponentMarketplacePage.tsx    # Page wrapper with provider
├── data/
│   └── marketplaceComponents.ts        # Data definitions and types
└── AppRoutes.tsx                       # Route configuration
```

## Features

### 1. **Search & Discovery**
- Real-time search with debouncing (300ms)
- Searches across component name, description, and tags
- Results update instantly as user types

### 2. **Category Filtering**
- Multi-category browsing with item counts
- All, Analytics, Trading, Visualization, Collaboration categories
- Active category highlighting

### 3. **Advanced Filtering**
- Price filtering (Free vs. Paid)
- Sorting options (Downloads, Rating, Alphabetical)
- Real-time filter application

### 4. **Featured Components**
- Curated selection displayed on initial load
- Only shown when viewing all categories with no search
- Eye-catching gradient design

### 5. **Component Details**
- Interactive modal with comprehensive information
- Shows:
  - Component metadata (author, version, price)
  - Statistics (rating, reviews, downloads)
  - Tags and dependencies
  - Installation instructions
  - Configuration examples
  - Preview, share, and favorite actions

### 6. **Installation Management**
- Install/Uninstall functionality
- Visual feedback for installed components
- Installation counter in header

### 7. **Accessibility**
- ARIA labels and roles
- Keyboard navigation support
- Screen reader friendly
- Semantic HTML structure

### 8. **Responsive Design**
- Mobile-friendly layout
- Adaptive grid (1 col mobile, 4 col desktop)
- Touch-friendly buttons and interactions

## File Structure

### `contexts/MarketplaceContext.tsx`
Manages all marketplace state:
- `searchQuery`: Current search input
- `selectedCategory`: Active category
- `selectedComponent`: Currently viewed component
- `installedComponents`: Set of installed component IDs
- `sortBy`: Sort preference
- `priceFilter`: Price range filter

**Key Methods:**
- `handleInstall(componentId)`: Install a component
- `handleUninstall(componentId)`: Remove a component
- `isInstalled(componentId)`: Check installation status

**Usage:**
```typescript
import { useMarketplace } from '@/contexts/MarketplaceContext';

function MyComponent() {
  const { 
    searchQuery, 
    installedComponents, 
    handleInstall 
  } = useMarketplace();
  // ...
}
```

### `components/marketplace/ComponentMarketplace.tsx`
Main orchestrator component:
- Integrates all subcomponents
- Handles filtering and sorting logic
- Manages layout and spacing
- Uses `useMemo` for performance optimization

**Key Dependencies:**
- SearchBar
- CategoryFilter
- ComponentCard
- ComponentModal
- FeaturedComponents

### `components/marketplace/ComponentCard.tsx`
Individual component card in the grid:
- Displays icon, name, description
- Shows rating and download count
- Install/Uninstall button
- Clickable to open detail modal
- Handles event propagation correctly

**Props:**
```typescript
interface ComponentCardProps {
  component: Component;
  isInstalled: boolean;
  onInstall: (componentId: string) => void;
  onUninstall: (componentId: string) => void;
  onSelect: (component: Component) => void;
}
```

### `components/marketplace/ComponentModal.tsx`
Detailed component information modal:
- Comprehensive metadata display
- Installation instructions
- Configuration examples
- Dependency list
- Share and preview actions
- Install/Uninstall controls

**Features:**
- Modal focus management
- Escape key to close
- Disabled preview button when no preview URL
- Share functionality (native or clipboard fallback)

### `components/marketplace/SearchBar.tsx`
Debounced search input:
- 300ms debounce delay (configurable)
- Prevents excessive re-renders
- Clear placeholder text
- Accessible search icon

**Props:**
```typescript
interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  debounceDelay?: number;  // Default: 300ms
}
```

### `components/marketplace/CategoryFilter.tsx`
Category selection component:
- Displays all available categories
- Shows item count per category
- Active category highlighting
- Accessible button group

### `components/marketplace/FeaturedComponents.tsx`
Featured components showcase:
- Grid of featured items
- Only displayed on empty search
- Eye-catching gradient design
- Clickable to view details

### `data/marketplaceComponents.ts`
Data definitions and types:
```typescript
export interface Component {
  id: string;
  name: string;
  description: string;
  category: string;
  author: string;
  downloads: number;
  rating: number;
  reviews: number;
  version: string;
  tags: string[];
  icon: string;
  price: 'free' | string;
  featured: boolean;
  preview?: string;
  dependencies: string[];
  config: ComponentConfig;
}

export interface Category {
  id: string;
  label: string;
  count: number;
}
```

**Data includes:**
- 9 sample components across categories
- Realistic metadata and statistics
- Installation instructions and configs
- Dependencies for each component

### `pages/marketplace/ComponentMarketplacePage.tsx`
Page wrapper component:
- Wraps ComponentMarketplace with MarketplaceProvider
- Provides centralized state management
- Ready for integration into routing

## Integration Points

### 1. **Menu Integration** (`MainNavigation.tsx`)
Added to Weave menu under new "Components" section:
```typescript
{
  label: 'Components',
  icon: <StoreIcon />,
  items: [
    { 
      label: 'Marketplace', 
      path: '/marketplace', 
      icon: <StoreIcon />,
      badge: { label: 'New', color: 'warning' } 
    },
    { 
      label: 'Custom Components', 
      path: '/fabric/custom-components' 
    },
  ]
}
```

### 2. **Route Configuration** (`AppRoutes.tsx`)
```typescript
<Route 
  path="/marketplace" 
  element={<ProtectedRoute><ComponentMarketplacePage /></ProtectedRoute>} 
/>
```

## Styling

### Color Scheme
- **Primary**: Slate palette (900-800-700-600-400)
- **Accent**: Blue (600-700) and Purple (400-500)
- **Success**: Green (600-700)
- **Warning**: Yellow (400)
- **Error**: Red (600-700)

### Tailwind Classes Used
- Grid layouts: `grid-cols-1`, `lg:grid-cols-4`
- Spacing: `gap-8`, `p-6`, `mb-4`
- Colors: Slate, Blue, Purple gradients
- Effects: `backdrop-blur`, `hover:`, `transition`
- Typography: Responsive sizing with `text-2xl`, `text-sm`

## State Management Flow

```
User Input
    ↓
MarketplaceContext
    ↓
[Search, Filter, Sort] Logic
    ↓
ComponentMarketplace (useMemo)
    ↓
Filter & Sort Components
    ↓
Render [Card | Modal]
```

## Performance Optimizations

1. **Memoization**
   - `useMemo` for filtered/sorted components
   - `useCallback` for handlers
   - Component memoization for modal

2. **Debouncing**
   - SearchBar debounces with 300ms delay
   - Reduces unnecessary re-renders

3. **Lazy Rendering**
   - Modal only renders when selected
   - Featured section hides on search
   - Truncation of tags with overflow indicators

4. **Efficient Filtering**
   - Single-pass filter logic
   - Conditional rendering of sections
   - No DOM mutations

## Accessibility Features

- ✅ ARIA labels on interactive elements
- ✅ Keyboard navigation support
- ✅ Semantic HTML structure
- ✅ Color contrast compliance
- ✅ Screen reader friendly text
- ✅ Focus management in modal
- ✅ Disabled states properly indicated

## Usage Examples

### Basic Setup
```typescript
// In your page component
import { MarketplaceProvider } from '@/contexts/MarketplaceContext';
import ComponentMarketplace from '@/components/marketplace/ComponentMarketplace';

function MyPage() {
  return (
    <MarketplaceProvider>
      <ComponentMarketplace />
    </MarketplaceProvider>
  );
}
```

### With Custom Initial State
```typescript
<MarketplaceProvider initialInstalledComponents={['Component1', 'Component2']}>
  <ComponentMarketplace />
</MarketplaceProvider>
```

### Accessing Context in Child Components
```typescript
import { useMarketplace } from '@/contexts/MarketplaceContext';

function MyComponent() {
  const { 
    searchQuery, 
    selectedCategory,
    installedComponents,
    handleInstall,
    handleUninstall
  } = useMarketplace();

  return (
    <div>
      {/* Use state and handlers */}
    </div>
  );
}
```

## Future Enhancement Opportunities

1. **Backend Integration**
   - Connect to API for real component data
   - Dynamic component loading
   - User installation history

2. **Advanced Features**
   - Component ratings and reviews UI
   - Dependency resolution
   - Auto-installation of dependencies
   - Component update notifications

3. **Analytics**
   - Track popular components
   - Installation analytics
   - Search trending

4. **User Preferences**
   - Save favorite components
   - Custom component collections
   - Recommended components based on usage

5. **Component Publishing**
   - Build component publishing workflow
   - Version management
   - Component marketplace publishing guidelines

## Troubleshooting

### Components Not Appearing
- Check `data/marketplaceComponents.ts` exports
- Verify MarketplaceProvider wraps the component
- Ensure route is properly configured in AppRoutes

### Search Not Working
- Verify debounce delay isn't too long
- Check component data for searchable fields
- Ensure SearchBar onChange handler is wired

### Installation Not Persisting
- State is in-memory; persist to localStorage/backend as needed
- Modify MarketplaceContext to add persistence layer
- Consider Redux or Zustand for complex state

## Testing

### Test Cases to Implement
1. Search filtering accuracy
2. Category filtering
3. Install/Uninstall functionality
4. Modal open/close
5. Responsive design across breakpoints
6. Keyboard navigation
7. Accessibility compliance

### Example Test
```typescript
test('filters components by search query', () => {
  render(
    <MarketplaceProvider>
      <ComponentMarketplace />
    </MarketplaceProvider>
  );
  
  const searchInput = screen.getByPlaceholderText('Search components...');
  fireEvent.change(searchInput, { target: { value: 'attribution' } });
  
  expect(screen.getByText('Attribution Analysis')).toBeInTheDocument();
});
```

## Related Files

- **Navigation**: `/frontend/src/components/MainNavigation.tsx`
- **Routing**: `/frontend/src/AppRoutes.tsx`
- **Types**: `/frontend/src/types/` (add marketplace types as needed)
- **Hooks**: `/frontend/src/hooks/` (custom hooks for marketplace logic)

## Conclusion

The Component Marketplace provides a solid foundation for component discovery and management. Its modular architecture, strong state management, and accessibility compliance make it a reliable feature for the Weave system. The codebase is well-positioned for future enhancements and backend integration.
