import React, { useState } from 'react';
import { useMutation } from '@apollo/client';
import { UPLOAD_DOCUMENT } from '../../graphql/ragQueries';

export const DocumentUpload: React.FC = () => {
  const [filePath, setFilePath] = useState('');
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState<string | null>(null);
  
  const [uploadDocument, { loading, error }] = useMutation(UPLOAD_DOCUMENT);

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!filePath.trim() || !title.trim()) return;

    try {
      const { data } = await uploadDocument({
        variables: { filePath, title },
      });
      setStatus(`Upload successful! Job ID: ${data.uploadDocument.document_id}`);
      setFilePath('');
      setTitle('');
    } catch (err) {
      console.error('Upload failed:', err);
      setStatus('Upload failed. Please try again.');
    }
  };

  return (
    <div className="p-6 max-w-2xl mx-auto bg-white rounded-xl shadow-md border border-gray-100">
      <h2 className="text-xl font-bold mb-6 text-gray-800">Upload Document</h2>
      
      <form onSubmit={handleUpload} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Document Title
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="e.g., Q3 Financial Report"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            File Path (Local Simulation)
          </label>
          <input
            type="text"
            value={filePath}
            onChange={(e) => setFilePath(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="/path/to/document.pdf"
          />
          <p className="text-xs text-gray-500 mt-1">
            * In a real app, this would be a file picker uploading to S3/Blob Storage.
          </p>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full py-2 px-4 bg-green-600 text-white font-medium rounded-md hover:bg-green-700 disabled:opacity-50 transition-colors"
        >
          {loading ? 'Processing...' : 'Start Ingestion'}
        </button>
      </form>

      {status && (
        <div className={`mt-4 p-3 rounded-md text-sm ${status.includes('failed') ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'}`}>
          {status}
        </div>
      )}
      
      {error && (
        <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md text-sm">
          Error: {error.message}
        </div>
      )}
    </div>
  );
};
