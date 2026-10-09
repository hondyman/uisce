import React from 'react';
import { registerDomainComponents } from '../../studio-core/components/registry';
import AppearanceSettings from '../../components/AppearanceSettings';

/**
 * Settings (the fabric-settings page). Appearance is saved per browser, so it is
 * the one real setting on the page; it is placed as the app's own chooser.
 */
registerDomainComponents([
  {
    id: 'platformSettings.Appearance',
    domain: 'platform_settings',
    label: 'Appearance',
    description: 'The style and colour mode for this browser.',
    inputs: [],
    events: [],
    render: () => <AppearanceSettings />,
  },
]);
