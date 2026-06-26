import React, { useState } from "react";
import { GenUIRenderer } from "./Renderer";
import { useGenUIIntent } from "./hooks";
import { Search, Loader2 } from "lucide-react";

/**
 * Example GenUI page with natural language query
 */
export function GenUIDashboardPage() {
  const [query, setQuery] = useState("");
  const { mutate: generateLayout, data, isLoading } = useGenUIIntent();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      generateLayout({
        query: query.trim(),
        tenant_id: "default",
        user_id: "current_user",
      });
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold text-gray-900 mb-6">
          AI-Powered Dashboard Builder
        </h1>

        {/* Query Input */}
        <form onSubmit={handleSubmit} className="mb-8">
          <div className="flex gap-2">
            <div className="flex-1 relative">
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Ask anything... e.g., 'Show portfolio performance over the last year'"
                className="w-full px-4 py-3 pl-12 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              <Search className="absolute left-4 top-3.5 w-5 h-5 text-gray-400" />
            </div>
            
            <button
              type="submit"
              disabled={isLoading || !query.trim()}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
            >
              {isLoading && <Loader2 className="w-4 h-4 animate-spin" />}
              Generate
            </button>
          </div>
        </form>

        {/* Intent Display */}
        {data?.intent && (
          <div className="mb-6 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="text-sm text-blue-900">
              <strong>Understood:</strong> {data.intent.type} with{" "}
              {data.intent.objects.join(", ")} (
              {(data.intent.confidence * 100).toFixed(0)}% confidence)
            </div>
          </div>
        )}

        {/* Dynamic Dashboard */}
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
          </div>
        ) : data?.layout ? (
          <GenUIRenderer layoutJson={data.layout} />
        ) : (
          <div className="text-center text-gray-500 py-12">
            <p>Ask a question to generate your dashboard</p>
            <div className="mt-4 space-y-2 text-sm">
              <p>Try:</p>
              <ul className="space-y-1">
                <li>"Show portfolio performance over time"</li>
                <li>"Display top holdings in a table"</li>
                <li>"Compare account values"</li>
              </ul>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
