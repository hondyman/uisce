import React, { useState, useEffect } from 'react';
import { offboardingApi, OffboardingRecord } from '../../api/offboardingApi';
import { UserMinus, RotateCcw, AlertCircle, CheckCircle2 } from 'lucide-react';
import { useForm } from 'react-hook-form';

export const AdminOffboardingPage: React.FC = () => {
  const [records, setRecords] = useState<OffboardingRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const { register, handleSubmit, reset } = useForm<{ user_id: string; reassign_to_user_id: string; reason: string }>();

  useEffect(() => {
    loadRecords();
  }, []);

  const loadRecords = async () => {
    setLoading(true);
    try {
      const { offboardings } = await offboardingApi.listOffboardings();
      setRecords(offboardings || []);
    } catch (error) {
       console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const showToast = (type: 'success' | 'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 4000);
  };

  const onSubmit = async (data: any) => {
    try {
      await offboardingApi.offboardUser(data);
      showToast('success', 'Offboarding initiated');
      reset();
      loadRecords();
    } catch (error) {
      showToast('error', 'Failed to offboard user');
    }
  };

  const handleReverse = async (id: string) => {
    if (!confirm('Are you sure you want to reverse this offboarding? Tasks will be returned.')) return;
    try {
      await offboardingApi.reverseOffboarding(id);
      showToast('success', 'Offboarding reversed');
      loadRecords();
    } catch (error) {
      showToast('error', 'Failed to reverse offboarding');
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-6 font-display">
      <div className="max-w-6xl mx-auto space-y-6">
        <header className="flex justify-between items-center pb-6 border-b border-gray-200 dark:border-gray-800">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
              <UserMinus className="w-8 h-8 text-red-500" />
              Employee Offboarding
            </h1>
            <p className="text-gray-500 dark:text-gray-400 mt-1">Manage user exits and asset reassignment</p>
          </div>
        </header>

        {toast && (
          <div className={`p-4 rounded-lg flex items-center gap-3 ${
            toast.type === 'success' ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'
          }`}>
            {toast.type === 'success' ? <CheckCircle2 className="w-5 h-5"/> : <AlertCircle className="w-5 h-5"/>}
            {toast.message}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* New Offboarding Form */}
          <div className="bg-white dark:bg-[#1c2127] p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 h-fit">
            <h2 className="text-lg font-bold text-gray-900 dark:text-white mb-4">Initiate Offboarding</h2>
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">User ID to Offboard</label>
                <input
                  {...register('user_id', { required: true })}
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-red-500"
                  placeholder="e.g. johndoe@company.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reassign Assets To</label>
                <input
                  {...register('reassign_to_user_id', { required: true })}
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g. manager@company.com"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reason</label>
                <textarea
                  {...register('reason', { required: true })}
                  rows={3}
                  className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                  placeholder="Reason for offboarding..."
                />
              </div>
              <button
                type="submit"
                className="w-full py-2 px-4 bg-red-600 hover:bg-red-700 text-white font-medium rounded-lg transition-colors flex justify-center items-center gap-2"
              >
                <UserMinus className="w-4 h-4" />
                Offboard User
              </button>
            </form>
          </div>

          {/* Records List */}
          <div className="lg:col-span-2 bg-white dark:bg-[#1c2127] rounded-xl shadow-sm border border-gray-200 dark:border-gray-800 overflow-hidden">
             <div className="p-4 border-b border-gray-200 dark:border-gray-800 flex justify-between items-center">
                <h2 className="text-lg font-bold text-gray-900 dark:text-white">History</h2>
                <button onClick={loadRecords} className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full">
                  <RotateCcw className="w-4 h-4 text-gray-500" />
                </button>
             </div>
             
             {loading ? (
                <div className="p-8 text-center text-gray-500">Loading...</div>
             ) : records.length === 0 ? (
                <div className="p-8 text-center text-gray-500">No offboarding records found.</div>
             ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left">
                    <thead className="bg-gray-50 dark:bg-[#111418] text-gray-600 dark:text-gray-400 text-xs uppercase font-semibold">
                      <tr>
                        <th className="px-6 py-3">Offboarded User</th>
                        <th className="px-6 py-3">Reassigned To</th>
                        <th className="px-6 py-3">Status</th>
                        <th className="px-6 py-3">Date</th>
                        <th className="px-6 py-3 text-right">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-800">
                      {records.map(r => (
                        <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                          <td className="px-6 py-4">
                            <div className="text-sm font-medium text-gray-900 dark:text-white">{r.user_id}</div>
                            <div className="text-xs text-gray-500">{r.reason}</div>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-600 dark:text-gray-300">{r.reassigned_to}</td>
                          <td className="px-6 py-4">
                            <span className={`inline-flex px-2 py-1 rounded-full text-xs font-semibold ${
                              r.status === 'COMPLETED' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                            }`}>
                              {r.status}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-sm text-gray-500">{new Date(r.created_at).toLocaleDateString()}</td>
                          <td className="px-6 py-4 text-right">
                             <button
                               onClick={() => handleReverse(r.id)}
                               className="text-gray-400 hover:text-red-500 transition-colors text-sm font-medium"
                             >
                               Undo
                             </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
             )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default AdminOffboardingPage;
