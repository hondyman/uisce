import React, { useState, useEffect } from 'react';
import { FileText, Upload, Download, Eye, CheckCircle, AlertCircle, Clock } from 'lucide-react';
import { fetchAPI } from '../../api';

// Types
interface AltInvestmentDocument {
  document_id: string;
  investment_id: string;
  document_type: 'K1' | 'CAPITAL_STATEMENT' | 'QUARTERLY_REPORT' | 'ANNUAL_REPORT' | 'SUBSCRIPTION_AGREEMENT' | 'OPERATING_AGREEMENT' | 'SIDE_LETTER' | 'OTHER';
  document_date: string | null;
  tax_year: number | null;
  file_url: string;
  file_name: string;
  extraction_status: 'PENDING' | 'IN_PROGRESS' | 'COMPLETED' | 'FAILED' | 'MANUAL_REVIEW_REQUIRED';
  extraction_confidence: number | null;
  uploaded_at: string;
}

interface DocumentVaultProps {
  investmentId: string;
  fundName: string;
}

export const DocumentVault: React.FC<DocumentVaultProps> = ({ investmentId, fundName }) => {
  const [documents, setDocuments] = useState<AltInvestmentDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    loadDocuments();
  }, [investmentId]);

  const loadDocuments = async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<AltInvestmentDocument[]>(
        `/alternative-investments/${investmentId}/documents`
      );
      setDocuments(data);
    } catch (error) {
      console.error('Failed to load documents:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setUploading(true);
    try {
      // In production, upload to S3 first, then create document record
      const formData = new FormData();
      formData.append('file', file);
      
      // Simulate upload
      const fileUrl = `https://storage.semlayer.com/altinvest/${investmentId}/${file.name}`;
      
      await fetchAPI(`/alternative-investments/${investmentId}/documents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          document_type: detectDocumentType(file.name),
          file_url: fileUrl,
          file_name: file.name,
          file_size_bytes: file.size,
          mime_type: file.type,
        }),
      });

      await loadDocuments();
    } catch (error) {
      console.error('Failed to upload document:', error);
    } finally {
      setUploading(false);
    }
  };

  const detectDocumentType = (filename: string): string => {
    const lower = filename.toLowerCase();
    if (lower.includes('k-1') || lower.includes('k1')) return 'K1';
    if (lower.includes('quarterly')) return 'QUARTERLY_REPORT';
    if (lower.includes('annual')) return 'ANNUAL_REPORT';
    if (lower.includes('capital')) return 'CAPITAL_STATEMENT';
    return 'OTHER';
  };

  const getStatusBadge = (doc: AltInvestmentDocument) => {
    switch (doc.extraction_status) {
      case 'COMPLETED':
        return (
          <span className="flex items-center gap-1 px-2 py-1 bg-green-50 text-green-700 rounded-full text-xs">
            <CheckCircle size={12} />
            Processed
          </span>
        );
      case 'PENDING':
      case 'IN_PROGRESS':
        return (
          <span className="flex items-center gap-1 px-2 py-1 bg-blue-50 text-blue-700 rounded-full text-xs">
            <Clock size={12} />
            Processing
          </span>
        );
      case 'MANUAL_REVIEW_REQUIRED':
        return (
          <span className="flex items-center gap-1 px-2 py-1 bg-orange-50 text-orange-700 rounded-full text-xs">
            <AlertCircle size={12} />
            Review Needed
          </span>
        );
      case 'FAILED':
        return (
          <span className="flex items-center gap-1 px-2 py-1 bg-red-50 text-red-700 rounded-full text-xs">
            <AlertCircle size={12} />
            Failed
          </span>
        );
      default:
        return null;
    }
  };

  const getDocumentTypeName = (type: string) => {
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  if (loading) return <div className="p-8 text-center">Loading documents...</div>;

  return (
    <div className="space-y-6">
      {/* Header with Upload */}
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">Document Vault</h2>
          <p className="text-sm text-gray-500 mt-1">{fundName}</p>
        </div>
        <label className="px-4 py-2 bg-blue-600 text-white rounded-lg flex items-center gap-2 hover:bg-blue-700 cursor-pointer">
          <Upload size={16} />
          Upload Document
          <input 
            type="file" 
            className="hidden" 
            onChange={handleFileUpload}
            accept=".pdf,.xlsx,.xls,.docx,.doc"
            disabled={uploading}
          />
        </label>
      </div>

      {/* AI Processing Notice */}
      {documents.some(d => d.extraction_status === 'PENDING' || d.extraction_status === 'IN_PROGRESS') && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="flex items-center gap-3">
            <Clock className="text-blue-600" size={20} />
            <div>
              <h3 className="font-medium text-blue-900">AI Processing in Progress</h3>
              <p className="text-sm text-blue-700">
                Gemini AI is extracting data from your documents. This usually takes 1-2 minutes.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Document List */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Document</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Date</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Confidence</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Uploaded</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {documents.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-6 py-12 text-center text-gray-500">
                    <FileText className="mx-auto mb-2 text-gray-400" size={48} />
                    No documents uploaded yet
                  </td>
                </tr>
              ) : (
                documents.map((doc) => (
                  <tr key={doc.document_id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <FileText className="text-gray-400" size={20} />
                        <span className="font-medium text-gray-900">{doc.file_name || 'Unnamed Document'}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      <span className="px-2 py-1 bg-gray-100 rounded text-xs">
                        {getDocumentTypeName(doc.document_type)}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {doc.document_date 
                        ? new Date(doc.document_date).toLocaleDateString() 
                        : doc.tax_year || '-'}
                    </td>
                    <td className="px-6 py-4">
                      {getStatusBadge(doc)}
                    </td>
                    <td className="px-6 py-4 text-sm">
                      {doc.extraction_confidence !== null ? (
                        <div className="flex items-center gap-2">
                          <div className="flex-1 bg-gray-200 rounded-full h-2 w-20">
                            <div 
                              className={`h-2 rounded-full ${
                                doc.extraction_confidence >= 0.8 ? 'bg-green-500' :
                                doc.extraction_confidence >= 0.5 ? 'bg-yellow-500' :
                                'bg-red-500'
                              }`}
                              style={{ width: `${doc.extraction_confidence * 100}%` }}
                            />
                          </div>
                          <span className="text-xs text-gray-600">
                            {(doc.extraction_confidence * 100).toFixed(0)}%
                          </span>
                        </div>
                      ) : (
                        <span className="text-gray-400">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-500">
                      {new Date(doc.uploaded_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <div className="flex justify-end gap-2">
                        <button className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
                          <Eye size={16} className="text-gray-600" />
                        </button>
                        <button className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
                          <Download size={16} className="text-gray-600" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Document Type Summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {['K1', 'QUARTERLY_REPORT', 'ANNUAL_REPORT', 'CAPITAL_STATEMENT'].map((type) => {
          const count = documents.filter(d => d.document_type === type).length;
          return (
            <div key={type} className="bg-white p-4 rounded-lg border border-gray-200">
              <p className="text-sm text-gray-500">{getDocumentTypeName(type)}</p>
              <p className="text-2xl font-bold text-gray-900 mt-1">{count}</p>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export default DocumentVault;
