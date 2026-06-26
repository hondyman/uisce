import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Edit2, Trash2, Save, X } from "lucide-react";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

interface BusinessObject {
  id: string;
  tenant_id: string;
  name: string;
  storage: string;
  version: number;
  status: string;
  fields: FieldDefinition[];
}

interface FieldDefinition {
  id: string;
  name: string;
  label: string;
  type: string;
  is_required: boolean;
  is_unique: boolean;
}

export function MetadataAdminPage() {
  const [selectedObject, setSelectedObject] = useState<BusinessObject | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const queryClient = useQueryClient();

  // Fetch business objects
  const { data: objects, isLoading } = useQuery<BusinessObject[]>({
    queryKey: ["metadata", "business-objects"],
    queryFn: async () => {
      const res = await fetch(`${API_BASE_URL}/meta/business-objects?tenant_id=default`);
      if (!res.ok) throw new Error("Failed to fetch");
      return res.json();
    },
  });

  // Create/Update business object
  const saveMutation = useMutation({
    mutationFn: async (bo: BusinessObject) => {
      const url = bo.id
        ? `${API_BASE_URL}/meta/business-objects/${bo.id}`
        : `${API_BASE_URL}/meta/business-objects`;
      
      const res = await fetch(url, {
        method: bo.id ? "PUT" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(bo),
      });

      if (!res.ok) throw new Error("Failed to save");
      return res.json();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["metadata", "business-objects"] });
      setSelectedObject(null);
      setIsCreating(false);
    },
  });

  // Delete business object
  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`${API_BASE_URL}/meta/business-objects/${id}`, {
        method: "DELETE",
      });
      if (!res.ok) throw new Error("Failed to delete");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["metadata", "business-objects"] });
    },
  });

  // Generate Hasura metadata
  const generateHasuraMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`${API_BASE_URL}/meta/business-objects/${id}/hasura`, {
        method: "POST",
      });
      if (!res.ok) throw new Error("Failed to generate");
      return res.json();
    },
  });

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold">Metadata Configuration</h1>
          <button
            onClick={() => {
              setIsCreating(true);
              setSelectedObject({
                id: "",
                tenant_id: "default",
                name: "",
                storage: "row",
                version: 1,
                status: "draft",
                fields: [],
              });
            }}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Plus className="w-4 h-4" />
            New Business Object
          </button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Object List */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-lg shadow">
              <div className="p-4 border-b">
                <h2 className="font-semibold">Business Objects</h2>
              </div>
              <div className="divide-y">
                {isLoading ? (
                  <div className="p-4 text-center text-gray-500">Loading...</div>
                ) : objects?.length === 0 ? (
                  <div className="p-4 text-center text-gray-500">No objects yet</div>
                ) : (
                  objects?.map((obj) => (
                    <div
                      key={obj.id}
                      className={`p-4 cursor-pointer hover:bg-gray-50 ${
                        selectedObject?.id === obj.id ? "bg-blue-50" : ""
                      }`}
                      onClick={() => setSelectedObject(obj)}
                    >
                      <div className="flex justify-between items-start">
                        <div>
                          <div className="font-medium">{obj.name}</div>
                          <div className="text-sm text-gray-500">
                            {obj.fields.length} fields · v{obj.version}
                          </div>
                        </div>
                        <span
                          className={`text-xs px-2 py-1 rounded ${
                            obj.status === "active"
                              ? "bg-green-100 text-green-800"
                              : "bg-gray-100 text-gray-800"
                          }`}
                        >
                          {obj.status}
                        </span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Object Editor */}
          <div className="lg:col-span-2">
            {selectedObject ? (
              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex justify-between items-center mb-6">
                  <h2 className="text-xl font-semibold">
                    {isCreating ? "New Business Object" : `Edit ${selectedObject.name}`}
                  </h2>
                  <button
                    onClick={() => {
                      setSelectedObject(null);
                      setIsCreating(false);
                    }}
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>

                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">Name</label>
                    <input
                      type="text"
                      value={selectedObject.name}
                      onChange={(e) =>
                        setSelectedObject({ ...selectedObject, name: e.target.value })
                      }
                      className="w-full border border-gray-300 rounded-lg px-3 py-2"
                    />
                  </div>

                  <div>
                    <label className="block text-sm font-medium mb-1">Storage Strategy</label>
                    <select
                      value={selectedObject.storage}
                      onChange={(e) =>
                        setSelectedObject({ ...selectedObject, storage: e.target.value })
                      }
                      className="w-full border border-gray-300 rounded-lg px-3 py-2"
                    >
                      <option value="row">Row-based</option>
                      <option value="wide_jsonb">Wide JSONB</option>
                      <option value="eav">EAV (Entity-Attribute-Value)</option>
                    </select>
                  </div>

                  <div className="flex gap-2">
                    <button
                      onClick={() => saveMutation.mutate(selectedObject)}
                      disabled={saveMutation.isPending}
                      className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                    >
                      <Save className="w-4 h-4" />
                      {saveMutation.isPending ? "Saving..." : "Save"}
                    </button>

                    {!isCreating && (
                      <>
                        <button
                          onClick={() => generateHasuraMutation.mutate(selectedObject.id)}
                          disabled={generateHasuraMutation.isPending}
                          className="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
                        >
                          {generateHasuraMutation.isPending
                            ? "Generating..."
                            : "Generate Hasura Metadata"}
                        </button>

                        <button
                          onClick={() => deleteMutation.mutate(selectedObject.id)}
                          disabled={deleteMutation.isPending}
                          className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                        >
                          <Trash2 className="w-4 h-4" />
                          Delete
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="bg-white rounded-lg shadow p-12 text-center text-gray-500">
                Select a business object or create a new one
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
