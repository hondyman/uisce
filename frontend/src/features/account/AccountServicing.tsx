import React, { useState } from 'react';
import { User, Lock, Mail, Phone, MapPin, Users, Shield, CreditCard, ArrowRight } from 'lucide-react';

export const AccountServicing: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'profile' | 'beneficiaries' | 'transactions'>('profile');

  return (
    <div className="max-w-6xl mx-auto p-6">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Account Settings</h1>
        <p className="text-gray-600">Manage your profile, beneficiaries, and account preferences</p>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-4 border-b border-gray-200 mb-8">
        <TabButton
          icon={<User />}
          label="Profile & Security"
          active={activeTab === 'profile'}
          onClick={() => setActiveTab('profile')}
        />
        <TabButton
          icon={<Users />}
          label="Beneficiaries"
          active={activeTab === 'beneficiaries'}
          onClick={() => setActiveTab('beneficiaries')}
        />
        <TabButton
          icon={<CreditCard />}
          label="Transactions"
          active={activeTab === 'transactions'}
          onClick={() => setActiveTab('transactions')}
        />
      </div>

      {/* Tab Content */}
      {activeTab === 'profile' && <ProfileSettings />}
      {activeTab === 'beneficiaries' && <BeneficiaryManagement />}
      {activeTab === 'transactions' && <TransactionRequests />}
    </div>
  );
};

const TabButton: React.FC<{ icon: React.ReactNode; label: string; active: boolean; onClick: () => void }> = ({
  icon,
  label,
  active,
  onClick,
}) => (
  <button
    onClick={onClick}
    className={`flex items-center gap-2 px-4 py-3 border-b-2 transition-colors ${
      active
        ? 'border-indigo-600 text-indigo-600 font-semibold'
        : 'border-transparent text-gray-600 hover:text-gray-900'
    }`}
  >
    {icon}
    {label}
  </button>
);

const ProfileSettings: React.FC = () => {
  const [profile, setProfile] = useState({
    firstName: 'John',
    lastName: 'Doe',
    email: 'john.doe@example.com',
    phone: '(555) 123-4567',
    address: '123 Main St',
    city: 'San Francisco',
    state: 'CA',
    zip: '94102',
  });

  const [isSaving, setIsSaving] = useState(false);

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await fetch('/api/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(profile),
      });
      alert('Profile updated successfully!');
    } catch (error) {
      console.error('Failed to update profile:', error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Personal Information</h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">First Name</label>
            <input
              type="text"
              value={profile.firstName}
              onChange={(e) => setProfile({ ...profile, firstName: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Last Name</label>
            <input
              type="text"
              value={profile.lastName}
              onChange={(e) => setProfile({ ...profile, lastName: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Email</label>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="email"
                value={profile.email}
                onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Phone</label>
            <div className="relative">
              <Phone className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
              <input
                type="tel"
                value={profile.phone}
                onChange={(e) => setProfile({ ...profile, phone: e.target.value })}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>
          </div>
        </div>

        <div className="mt-6">
          <label className="block text-sm font-medium text-gray-700 mb-2">Address</label>
          <input
            type="text"
            value={profile.address}
            onChange={(e) => setProfile({ ...profile, address: e.target.value })}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent mb-3"
          />

          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2">
              <input
                type="text"
                placeholder="City"
                value={profile.city}
                onChange={(e) => setProfile({ ...profile, city: e.target.value })}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
            </div>
            <input
              type="text"
              placeholder="State"
              value={profile.state}
              onChange={(e) => setProfile({ ...profile, state: e.target.value })}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              maxLength={2}
            />
          </div>
        </div>

        <button
          onClick={handleSave}
          disabled={isSaving}
          className="mt-6 px-6 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:from-indigo-700 hover:to-purple-700 disabled:opacity-50 transition-all font-medium"
        >
          {isSaving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <div className="flex items-center gap-3 mb-4">
          <Shield className="w-6 h-6 text-indigo-600" />
          <h3 className="text-lg font-semibold text-gray-900">Security Settings</h3>
        </div>

        <div className="space-y-4">
          <button className="w-full flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <div className="flex items-center gap-3">
              <Lock className="w-5 h-5 text-gray-600" />
              <div className="text-left">
                <p className="font-medium text-gray-900">Change Password</p>
                <p className="text-sm text-gray-600">Last changed 3 months ago</p>
              </div>
            </div>
            <ArrowRight className="w-5 h-5 text-gray-400" />
          </button>

          <button className="w-full flex items-center justify-between p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
            <div className="flex items-center gap-3">
              <Shield className="w-5 h-5 text-gray-600" />
              <div className="text-left">
                <p className="font-medium text-gray-900">Two-Factor Authentication</p>
                <p className="text-sm text-green-600">✓ Enabled</p>
              </div>
            </div>
            <ArrowRight className="w-5 h-5 text-gray-400" />
          </button>
        </div>
      </div>
    </div>
  );
};

const BeneficiaryManagement: React.FC = () => {
  const [beneficiaries, setBeneficiaries] = useState([
    { id: '1', name: 'Jane Doe', relationship: 'Spouse', allocation: 50 },
    { id: '2', name: 'John Doe Jr.', relationship: 'Child', allocation: 50 },
  ]);

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl border border-gray-200 p-6">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold text-gray-900">Primary Beneficiaries</h3>
          <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors text-sm font-medium">
            Add Beneficiary
          </button>
        </div>

        <div className="space-y-4">
          {beneficiaries.map((ben) => (
            <div key={ben.id} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
              <div>
                <p className="font-medium text-gray-900">{ben.name}</p>
                <p className="text-sm text-gray-600">{ben.relationship}</p>
              </div>
              <div className="text-right">
                <p className="font-semibold text-gray-900">{ben.allocation}%</p>
                <button className="text-sm text-indigo-600 hover:text-indigo-700">Edit</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

const TransactionRequests: React.FC = () => {
  const [amount, setAmount] = useState('');
  const [type, setType] = useState<'CONTRIBUTION' | 'WITHDRAWAL'>('CONTRIBUTION');

  const submitRequest = async () => {
    try {
      await fetch('/api/transactions/request', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, amount: parseFloat(amount) }),
      });
      alert('Transaction request submitted!');
      setAmount('');
    } catch (error) {
      console.error('Failed to submit request:', error);
    }
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-6">Request Transaction</h3>

      <div className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Transaction Type</label>
          <div className="grid grid-cols-2 gap-4">
            <button
              onClick={() => setType('CONTRIBUTION')}
              className={`p-4 border-2 rounded-lg transition-all ${
                type === 'CONTRIBUTION'
                  ? 'border-indigo-600 bg-indigo-50 text-indigo-700'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <p className="font-semibold">Contribution</p>
              <p className="text-sm text-gray-600">Add funds</p>
            </button>

            <button
              onClick={() => setType('WITHDRAWAL')}
              className={`p-4 border-2 rounded-lg transition-all ${
                type === 'WITHDRAWAL'
                  ? 'border-indigo-600 bg-indigo-50 text-indigo-700'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
            >
              <p className="font-semibold">Withdrawal</p>
              <p className="text-sm text-gray-600">Withdraw funds</p>
            </button>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Amount</label>
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="$0.00"
            className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent text-lg"
          />
        </div>

        <button
          onClick={submitRequest}
          disabled={!amount}
          className="w-full px-6 py-3 bg-gradient-to-r from-indigo-600 to-purple-600 text-white rounded-lg hover:from-indigo-700 hover:to-purple-700 disabled:opacity-50 transition-all font-medium"
        >
          Submit Request
        </button>

        <p className="text-sm text-gray-600">
          Your advisor will review and approve this request within 1-2 business days.
        </p>
      </div>
    </div>
  );
};
