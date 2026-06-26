/**
 * Component Marketplace Page
 * Wraps the marketplace with provider context
 */

import type React from 'react';
import { MarketplaceProvider } from '../../contexts/MarketplaceContext';
import ComponentMarketplace from '../../components/marketplace/ComponentMarketplace';

const ComponentMarketplacePage: React.FC = () => {
  return (
    <MarketplaceProvider>
      <ComponentMarketplace />
    </MarketplaceProvider>
  );
};

export default ComponentMarketplacePage;
