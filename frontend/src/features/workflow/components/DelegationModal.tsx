import React from 'react';
import { useForm } from 'react-hook-form';
import { X } from 'lucide-react';
import { delegationApi, DelegationRequest } from '../../../api/delegationApi';

interface DelegationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export const DelegationModal: React.FC<DelegationModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const { register, handleSubmit, formState: { errors, isSubmitting }, reset } = useForm<DelegationRequest>();

  const onSubmit = async (data: DelegationRequest) => {
    try {
      // Ensure arrays if string is passed (simplified input for now, ideally multi-select)
      const formattedData = {
        ...data,
        roles: typeof data.roles === 'string' ? (data.roles as string).split(',').map((s: string) => s.trim()) : data.roles,
        workflows: typeof data.workflows === 'string' ? (data.workflows as string).split(',').map((s: string) => s.trim()) : data.workflows,
      };

      await delegationApi.createDelegation(formattedData);
      reset();
      onSuccess();
      onClose();
    } catch (error) {
      console.error('Failed to create delegation:', error);
      // Ideally show error toast
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="w-full max-w-md bg-white dark:bg-[#1c2127] rounded-lg shadow-xl border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-bold text-gray-900 dark:text-white">New Delegation</h2>
          <button onClick={onClose} className="p-1 rounded-full hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-500 dark:text-gray-400">
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)} className="p-4 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Delegate To (User ID)</label>
            <input
              {...register('to_user_id', { required: 'User ID is required' })}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              placeholder="e.g. user@example.com"
            />
            {errors.to_user_id && <p className="text-red-500 text-xs mt-1">{errors.to_user_id.message}</p>}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Start Date</label>
              <input
                type="date"
                {...register('from_date', { required: 'Start date is required' })}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              />
              {errors.from_date && <p className="text-red-500 text-xs mt-1">{errors.from_date.message}</p>}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">End Date</label>
              <input
                type="date"
                {...register('to_date', { required: 'End date is required' })}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              />
              {errors.to_date && <p className="text-red-500 text-xs mt-1">{errors.to_date.message}</p>}
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reason</label>
            <textarea
              {...register('reason', { required: 'Reason is required' })}
              rows={3}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              placeholder="Why are you delegating?"
            />
            {errors.reason && <p className="text-red-500 text-xs mt-1">{errors.reason.message}</p>}
          </div>

          <div>
             <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Roles (comma separated)</label>
             <input
              {...register('roles')}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-[#111418] text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
              placeholder="Optional: specific roles to delegate"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg disabled:opacity-50"
            >
              {isSubmitting ? 'Creating...' : 'Create Delegation'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
