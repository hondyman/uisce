import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useABAC } from '../hooks/useABAC'

interface NamespaceConfig {
  id: string
  name: string
  description: string
  retentionPeriod: string
  maxWorkflowTimeout: string
  maxActivityTimeout: string
  policies: NamespacePolicy[]
}

interface NamespacePolicy {
  id: string
  name: string
  resource: string
  action: string
  effect: 'allow' | 'deny'
  conditions?: string
}

const fetchNamespaces = async () => {
  // Since backend doesn't have namespace CRUD yet, we'll simulate with tenant-specific data
  const res = await fetch('/api/tenant/namespaces')
  return res.ok ? res.json() : []
}

const saveNamespace = async (namespace: NamespaceConfig) => {
  const res = await fetch('/api/tenant/namespaces', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(namespace)
  })
  return res.json()
}

export default function NamespaceManager() {
  const { evaluate } = useABAC()
  const queryClient = useQueryClient()
  const [selectedNamespace, setSelectedNamespace] = useState<NamespaceConfig | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [newPolicy, setNewPolicy] = useState<Partial<NamespacePolicy>>({})

  const { data: namespaces = [], isLoading } = useQuery(['namespaces'], fetchNamespaces)

  const saveMutation = useMutation(saveNamespace, {
    onSuccess: () => {
      queryClient.invalidateQueries(['namespaces'])
      setIsEditing(false)
      setSelectedNamespace(null)
    }
  })

  // Check permissions on component mount
  useEffect(() => {
    const checkPermissions = async () => {
      const canView = await evaluate('read', 'namespaces')
      if (!canView) {
        alert('Access denied: Insufficient permissions to view namespaces')
      }
    }
    checkPermissions()
  }, [evaluate])

  const handleSave = async () => {
    if (!selectedNamespace) return

    const canCreate = await evaluate('create', 'namespace')
    const canUpdate = await evaluate('update', 'namespace')
    if (!canCreate && !canUpdate) {
      alert('Access denied: Insufficient permissions to manage namespaces')
      return
    }

    saveMutation.mutate(selectedNamespace)
  }

  const handleAddPolicy = async () => {
    if (!selectedNamespace || !newPolicy.name || !newPolicy.resource || !newPolicy.action) return

    const canUpdate = await evaluate('update', 'namespace-policies')
    if (!canUpdate) {
      alert('Access denied: Insufficient permissions to modify namespace policies')
      return
    }

    const policy: NamespacePolicy = {
      id: Date.now().toString(),
      name: newPolicy.name,
      resource: newPolicy.resource,
      action: newPolicy.action,
      effect: newPolicy.effect || 'allow',
      conditions: newPolicy.conditions
    }

    setSelectedNamespace({
      ...selectedNamespace,
      policies: [...selectedNamespace.policies, policy]
    })
    setNewPolicy({})
  }

  const handleRemovePolicy = async (policyId: string) => {
    if (!selectedNamespace) return

    const canUpdate = await evaluate('update', 'namespace-policies')
    if (!canUpdate) {
      alert('Access denied: Insufficient permissions to modify namespace policies')
      return
    }

    setSelectedNamespace({
      ...selectedNamespace,
      policies: selectedNamespace.policies.filter(p => p.id !== policyId)
    })
  }

  const createNewNamespace = () => {
    const newNs: NamespaceConfig = {
      id: '',
      name: '',
      description: '',
      retentionPeriod: '30d',
      maxWorkflowTimeout: '24h',
      maxActivityTimeout: '1h',
      policies: []
    }
    setSelectedNamespace(newNs)
    setIsEditing(true)
  }

  if (isLoading) {
    return <div className="p-4">Loading namespaces...</div>
  }

  return (
    <div className="p-4 max-w-6xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-semibold">Namespace Management</h2>
        <button
          onClick={createNewNamespace}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          Create Namespace
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Namespace List */}
        <div className="bg-white rounded-lg shadow">
          <div className="p-4 border-b">
            <h3 className="font-medium">Namespaces ({namespaces.length})</h3>
          </div>
          <div className="max-h-96 overflow-y-auto">
            {namespaces.map((ns: NamespaceConfig) => (
              <div
                key={ns.id}
                className={`p-3 border-b cursor-pointer hover:bg-gray-50 ${
                  selectedNamespace?.id === ns.id ? 'bg-blue-50 border-blue-200' : ''
                }`}
                onClick={() => {
                  setSelectedNamespace(ns)
                  setIsEditing(false)
                }}
              >
                <div className="font-medium">{ns.name}</div>
                <div className="text-sm text-gray-600">{ns.description}</div>
                <div className="text-xs text-gray-500 mt-1">
                  {ns.policies.length} policies
                </div>
              </div>
            ))}
            {namespaces.length === 0 && (
              <div className="p-8 text-center text-gray-500">
                No namespaces configured
              </div>
            )}
          </div>
        </div>

        {/* Namespace Details */}
        {selectedNamespace && (
          <div className="lg:col-span-2 bg-white rounded-lg shadow">
            <div className="p-4 border-b flex justify-between items-center">
              <h3 className="font-medium">
                {isEditing ? 'Edit Namespace' : 'Namespace Details'}
              </h3>
              <div className="flex gap-2">
                {!isEditing ? (
                  <button
                    onClick={() => setIsEditing(true)}
                    className="px-3 py-1 bg-gray-600 text-white rounded text-sm hover:bg-gray-700"
                  >
                    Edit
                  </button>
                ) : (
                  <>
                    <button
                      onClick={handleSave}
                      disabled={saveMutation.isLoading}
                      className="px-3 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700 disabled:bg-gray-400"
                    >
                      {saveMutation.isLoading ? 'Saving...' : 'Save'}
                    </button>
                    <button
                      onClick={() => {
                        setIsEditing(false)
                        setSelectedNamespace(null)
                      }}
                      className="px-3 py-1 bg-gray-600 text-white rounded text-sm hover:bg-gray-700"
                    >
                      Cancel
                    </button>
                  </>
                )}
              </div>
            </div>

            <div className="p-4 space-y-4">
              {/* Basic Settings */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Name
                  </label>
                  {isEditing ? (
                    <input
                      type="text"
                      value={selectedNamespace.name}
                      onChange={(e) => setSelectedNamespace({...selectedNamespace, name: e.target.value})}
                      className="w-full px-3 py-2 border rounded"
                      aria-label="Namespace name"
                    />
                  ) : (
                    <p className="text-sm">{selectedNamespace.name}</p>
                  )}
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Retention Period
                  </label>
                  {isEditing ? (
                    <input
                      type="text"
                      value={selectedNamespace.retentionPeriod}
                      onChange={(e) => setSelectedNamespace({...selectedNamespace, retentionPeriod: e.target.value})}
                      className="w-full px-3 py-2 border rounded"
                      placeholder="30d"
                      aria-label="Retention period"
                    />
                  ) : (
                    <p className="text-sm">{selectedNamespace.retentionPeriod}</p>
                  )}
                </div>
                <div className="md:col-span-2">
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Description
                  </label>
                  {isEditing ? (
                    <textarea
                      value={selectedNamespace.description}
                      onChange={(e) => setSelectedNamespace({...selectedNamespace, description: e.target.value})}
                      className="w-full px-3 py-2 border rounded"
                      rows={3}
                      aria-label="Namespace description"
                    />
                  ) : (
                    <p className="text-sm">{selectedNamespace.description}</p>
                  )}
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Max Workflow Timeout
                  </label>
                  {isEditing ? (
                    <input
                      type="text"
                      value={selectedNamespace.maxWorkflowTimeout}
                      onChange={(e) => setSelectedNamespace({...selectedNamespace, maxWorkflowTimeout: e.target.value})}
                      className="w-full px-3 py-2 border rounded"
                      placeholder="24h"
                      aria-label="Max workflow timeout"
                    />
                  ) : (
                    <p className="text-sm">{selectedNamespace.maxWorkflowTimeout}</p>
                  )}
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Max Activity Timeout
                  </label>
                  {isEditing ? (
                    <input
                      type="text"
                      value={selectedNamespace.maxActivityTimeout}
                      onChange={(e) => setSelectedNamespace({...selectedNamespace, maxActivityTimeout: e.target.value})}
                      className="w-full px-3 py-2 border rounded"
                      placeholder="1h"
                      aria-label="Max activity timeout"
                    />
                  ) : (
                    <p className="text-sm">{selectedNamespace.maxActivityTimeout}</p>
                  )}
                </div>
              </div>

              {/* Policies Section */}
              <div className="border-t pt-4">
                <h4 className="font-medium mb-3">ABAC Policies ({selectedNamespace.policies.length})</h4>

                {isEditing && (
                  <div className="bg-gray-50 p-3 rounded mb-4">
                    <h5 className="text-sm font-medium mb-2">Add New Policy</h5>
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mb-2">
                      <input
                        type="text"
                        placeholder="Policy name"
                        value={newPolicy.name || ''}
                        onChange={(e) => setNewPolicy({...newPolicy, name: e.target.value})}
                        className="px-2 py-1 border rounded text-sm"
                        aria-label="Policy name"
                      />
                      <input
                        type="text"
                        placeholder="Resource"
                        value={newPolicy.resource || ''}
                        onChange={(e) => setNewPolicy({...newPolicy, resource: e.target.value})}
                        className="px-2 py-1 border rounded text-sm"
                        aria-label="Resource"
                      />
                      <input
                        type="text"
                        placeholder="Action"
                        value={newPolicy.action || ''}
                        onChange={(e) => setNewPolicy({...newPolicy, action: e.target.value})}
                        className="px-2 py-1 border rounded text-sm"
                        aria-label="Action"
                      />
                      <select
                        value={newPolicy.effect || 'allow'}
                        onChange={(e) => setNewPolicy({...newPolicy, effect: e.target.value as 'allow' | 'deny'})}
                        className="px-2 py-1 border rounded text-sm"
                        aria-label="Policy effect"
                      >
                        <option value="allow">Allow</option>
                        <option value="deny">Deny</option>
                      </select>
                    </div>
                    <button
                      onClick={handleAddPolicy}
                      className="px-3 py-1 bg-blue-600 text-white rounded text-sm hover:bg-blue-700"
                    >
                      Add Policy
                    </button>
                  </div>
                )}

                <div className="space-y-2">
                  {selectedNamespace.policies.map((policy) => (
                    <div key={policy.id} className="flex items-center justify-between bg-gray-50 p-3 rounded">
                      <div className="flex-1">
                        <div className="font-medium text-sm">{policy.name}</div>
                        <div className="text-xs text-gray-600">
                          {policy.resource} : {policy.action} → {policy.effect.toUpperCase()}
                        </div>
                        {policy.conditions && (
                          <div className="text-xs text-gray-500 mt-1">
                            Conditions: {policy.conditions}
                          </div>
                        )}
                      </div>
                      {isEditing && (
                        <button
                          onClick={() => handleRemovePolicy(policy.id)}
                          className="px-2 py-1 bg-red-600 text-white rounded text-xs hover:bg-red-700"
                        >
                          Remove
                        </button>
                      )}
                    </div>
                  ))}
                  {selectedNamespace.policies.length === 0 && (
                    <div className="text-center text-gray-500 py-4">
                      No policies configured
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
