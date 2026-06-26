import React, { useState, useEffect } from 'react';
import { MetaRenderer } from '../../../components/metadata/MetaRenderer';
import { devDebug } from '../../../utils/devLogger';

// Sample metadata-driven Client view definition
const clientViewMetadata = {
  id: 'view_client_form',
  type: 'Form' as const,
  sections: [
    [
      { label: 'Client Code', attr: 'client_code', component: 'Text', required: true, helpText: 'Unique identifier for the client' },
      { label: 'Risk Tolerance', attr: 'risk_tolerance', component: 'Select', required: true },
    ],
    [
      { label: 'First Name', attr: 'first_name', component: 'Text', required: true },
      { label: 'Last Name', attr: 'last_name', component: 'Text', required: true },
    ],
    [
      { label: 'Date of Birth', attr: 'date_of_birth', component: 'Date', required: false },
      { label: 'Email', attr: 'email', component: 'Text', required: false },
    ],
    [
      { label: 'Net Worth', attr: 'net_worth', component: 'Number', required: false, helpText: 'Estimated net worth in USD' },
      { label: 'Annual Income', attr: 'annual_income', component: 'Number', required: false },
    ],
  ],
  dataSource: 'bo_client',
  actions: ['Save Client', 'Cancel'],
  theme: {
    font: 'Inter, system-ui, sans-serif',
    textColor: '#1f2937',
    gap: '16px',
  },
};

export const ClientManagementPage: React.FC = () => {
  const [clientData, setClientData] = useState<Record<string, any>>({
    client_code: '',
    first_name: '',
    last_name: '',
    risk_tolerance: 'MODERATE',
    date_of_birth: '',
    email: '',
    net_worth: '',
    annual_income: '',
  });

  const [clients, setClients] = useState<any[]>([]);
  const [selectedClientId, setSelectedClientId] = useState<string | null>(null);

  useEffect(() => {
    // Mock: Fetch clients from API
    // In production: fetch('/api/wealth/clients')
    setClients([
      { id: '1', client_code: 'CLI001', first_name: 'John', last_name: 'Doe', risk_tolerance: 'MODERATE' },
      { id: '2', client_code: 'CLI002', first_name: 'Jane', last_name: 'Smith', risk_tolerance: 'AGGRESSIVE' },
    ]);
  }, []);

  const handleFieldChange = (attr: string, value: any) => {
    setClientData((prev) => ({ ...prev, [attr]: value }));
  };

  const handleAction = async (action: string) => {
    if (action === 'Save Client') {
      // Mock: Save to API
      devDebug('Saving client:', clientData);
      alert('Client saved successfully! (Mock)');
      
      // In production:
      // const response = await fetch('/api/wealth/clients', {
      //   method: 'POST',
      //   headers: { 'Content-Type': 'application/json' },
      //   body: JSON.stringify(clientData),
      // });
      // if (response.ok) { ... }
    } else if (action === 'Cancel') {
      setClientData({
        client_code: '',
        first_name: '',
        last_name: '',
        risk_tolerance: 'MODERATE',
        date_of_birth: '',
        email: '',
        net_worth: '',
        annual_income: '',
      });
    }
  };

  const handleSelectClient = (client: any) => {
    setSelectedClientId(client.id);
    setClientData({
      client_code: client.client_code,
      first_name: client.first_name,
      last_name: client.last_name,
      risk_tolerance: client.risk_tolerance,
      date_of_birth: client.date_of_birth || '',
      email: client.email || '',
      net_worth: client.net_worth || '',
      annual_income: client.annual_income || '',
    });
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-900 mb-6">Client Management</h1>
        
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Client List */}
          <div className="bg-white rounded-lg shadow p-4">
            <h2 className="text-lg font-semibold mb-4 text-gray-800">Clients</h2>
            <div className="space-y-2">
              {clients.map((client) => (
                <div
                  key={client.id}
                  onClick={() => handleSelectClient(client)}
                  className={`p-3 rounded cursor-pointer transition-colors ${
                    selectedClientId === client.id
                      ? 'bg-blue-100 border-blue-500 border'
                      : 'bg-gray-50 hover:bg-gray-100'
                  }`}
                >
                  <p className="font-medium text-gray-900">{client.client_code}</p>
                  <p className="text-sm text-gray-600">{client.first_name} {client.last_name}</p>
                  <span className="text-xs bg-gray-200 px-2 py-1 rounded">{client.risk_tolerance}</span>
                </div>
              ))}
            </div>
            <button className="mt-4 w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700">
              + New Client
            </button>
          </div>

          {/* Metadata-Driven Form */}
          <div className="lg:col-span-2">
            <MetaRenderer
              view={clientViewMetadata}
              data={clientData}
              onChange={handleFieldChange}
              onAction={handleAction}
            />
          </div>
        </div>
      </div>
    </div>
  );
};