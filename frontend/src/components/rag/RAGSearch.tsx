import React, { useState } from 'react';
import { useMutation } from '@apollo/client';
import { SEARCH_RAG } from '../../graphql/ragQueries';

interface SearchResult {
  chunk_id: string;
  document_id: string;
  content: string;
  score: number;
  metadata: any;
}

export const RAGSearch: React.FC = () => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searchRAG, { loading, error }] = useMutation(SEARCH_RAG);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    try {
      const { data } = await searchRAG({
        variables: { query, limit: 5 },
      });
      setResults(data.searchRAG.results);
    } catch (err) {
      console.error('Search failed:', err);
    }
  };

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <h2 className="text-2xl font-bold mb-4 text-gray-800">Knowledge Base Search</h2>
      
      <form onSubmit={handleSearch} className="mb-8">
        <div className="flex gap-2">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Ask a question about your documents..."
            className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <button
            type="submit"
            disabled={loading}
            className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {loading ? 'Searching...' : 'Search'}
          </button>
        </div>
      </form>

      {error && (
        <div className="p-4 mb-4 text-red-700 bg-red-100 rounded-lg">
          Error: {error.message}
        </div>
      )}

      <div className="space-y-4">
        {results.map((result) => (
          <div key={result.chunk_id} className="p-4 bg-white border border-gray-200 rounded-lg shadow-sm hover:shadow-md transition-shadow">
            <div className="flex justify-between items-start mb-2">
              <span className="text-sm font-medium text-blue-600">
                Score: {(result.score * 100).toFixed(1)}%
              </span>
              <span className="text-xs text-gray-500">
                ID: {result.document_id.slice(0, 8)}...
              </span>
            </div>
            <p className="text-gray-700 leading-relaxed">{result.content}</p>
          </div>
        ))}
        
        {results.length === 0 && !loading && query && (
          <p className="text-center text-gray-500 mt-8">No results found.</p>
        )}
      </div>
    </div>
  );
};
