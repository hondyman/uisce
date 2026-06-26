import React, { useState, useCallback } from 'react';
import { Upload, FileText, CheckCircle, AlertCircle, Download, Eye, Trash2, Shield } from 'lucide-react';
import { useDropzone } from 'react-dropzone';

interface Document {
  documentId: string;
  documentType: string;
  fileName: string;
  fileUrl: string;
  fileSizeBytes: number;
  ocrExtractedData: any;
  ocrConfidence: number | null;
  verificationStatus: string;
  verificationNotes: string | null;
  uploadedAt: string;
}

const DOCUMENT_TYPES = {
  DRIVERS_LICENSE: { label: "Driver's License", icon: '🪪' },
  PASSPORT: { label: 'Passport', icon: '📘' },
  W9: { label: 'W-9 Tax Form', icon: '📄' },
  BANK_STATEMENT: { label: 'Bank Statement', icon: '🏦' },
  PROOF_OF_ADDRESS: { label: 'Proof of Address', icon: '🏠' },
  TAX_RETURN: { label: 'Tax Return', icon: '📊' },
  OTHER: { label: 'Other', icon: '📎' },
};

const VERIFICATION_STATUS = {
  PENDING: { label: 'Processing', color: 'text-yellow-600', bgColor: 'bg-yellow-100', icon: AlertCircle },
  IN_REVIEW: { label: 'In Review', color: 'text-blue-600', bgColor: 'bg-blue-100', icon: AlertCircle },
  VERIFIED: { label: 'Verified', color: 'text-green-600', bgColor: 'bg-green-100', icon: CheckCircle },
  REJECTED: { label: 'Rejected', color: 'text-red-600', bgColor: 'bg-red-100', icon: AlertCircle },
};

export const DocumentVaultUI: React.FC = () => {
  const [documents, setDocuments] = useState<Document[]>([]);
  const [selectedType, setSelectedType] = useState<string>('DRIVERS_LICENSE');
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({});

  useEffect(() => {
    fetchDocuments();
  }, []);

  const fetchDocuments = async () => {
    try {
      const response = await fetch('/api/documents');
      const data = await response.json();
      setDocuments(data);
    } catch (error) {
      console.error('Failed to fetch documents:', error);
    }
  };

  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    setIsUploading(true);

    for (const file of acceptedFiles) {
      try {
        // Upload to S3 or direct storage
        const formData = new FormData();
        formData.append('file', file);
        formData.append('documentType', selectedType);

        const uploadResponse = await fetch('/api/documents/upload', {
          method: 'POST',
          body: formData,
        });

        if (uploadResponse.ok) {
          const uploadData = await uploadResponse.json();
          
          // Create document record
          const docResponse = await fetch('/api/documents', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              documentType: selectedType,
              fileUrl: uploadData.fileUrl,
              fileName: file.name,
              fileSizeBytes: file.size,
              mimeType: file.type,
            }),
          });

          if (docResponse.ok) {
            const newDoc = await docResponse.json();
            setDocuments(prev => [newDoc, ...prev]);

            // Poll for OCR results
            pollOCRStatus(newDoc.documentId);
          }
        }
      } catch (error) {
        console.error('Upload failed:', error);
      }
    }

    setIsUploading(false);
  }, [selectedType]);

  const pollOCRStatus = async (documentId: string) => {
    const maxAttempts = 20;
    let attempts = 0;

    const poll = async () => {
      try {
        const response = await fetch(`/api/documents/${documentId}`);
        const doc = await response.json();

        // Update document in state
        setDocuments(prev => 
          prev.map(d => d.documentId === documentId ? doc : d)
        );

        // If still processing, continue polling
        if (doc.verificationStatus === 'PENDING' && attempts < maxAttempts) {
          attempts++;
          setTimeout(poll, 2000);
        }
      } catch (error) {
        console.error('Failed to poll OCR status:', error);
      }
    };

    poll();
  };

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'image/*': ['.png', '.jpg', '.jpeg'],
      'application/pdf': ['.pdf'],
    },
    maxSize: 10 * 1024 * 1024, // 10MB
  });

  const deleteDocument = async (documentId: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return;

    try {
      await fetch(`/api/documents/${documentId}`, { method: 'DELETE' });
      setDocuments(prev => prev.filter(d => d.documentId !== documentId));
    } catch (error) {
      console.error('Failed to delete document:', error);
    }
  };

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="max-w-6xl mx-auto p-6">
      <div className="mb-8">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Document Vault</h1>
            <p className="text-gray-600 mt-1">Securely store and manage your documents with AI-powered extraction</p>
          </div>
          <div className="flex items-center gap-2 text-sm text-green-600">
            <Shield className="w-5 h-5" />
            <span>Bank-level encryption</span>
          </div>
        </div>

        {/* Document Type Selector */}
        <div className="flex gap-2 mb-4 flex-wrap">
          {Object.entries(DOCUMENT_TYPES).map(([key, config]) => (
            <button
              key={key}
              onClick={() => setSelectedType(key)}
              className={`px-4 py-2 rounded-lg border-2 transition-all ${
                selectedType === key
                  ? 'border-indigo-600 bg-indigo-50 text-indigo-700 font-semibold'
                  : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300'
              }`}
            >
              <span className="mr-2">{config.icon}</span>
              {config.label}
            </button>
          ))}
        </div>

        {/* Upload Zone */}
        <div
          {...getRootProps()}
          className={`border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-all ${
            isDragActive
              ? 'border-indigo-500 bg-indigo-50'
              : 'border-gray-300 bg-gray-50 hover:border-indigo-400 hover:bg-indigo-50'
          }`}
        >
          <input {...getInputProps()} />
          <Upload className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          {isDragActive ? (
            <p className="text-lg text-indigo-600 font-medium">Drop files here...</p>
          ) : (
            <>
              <p className="text-lg text-gray-700 font-medium mb-2">
                Drag & drop your {DOCUMENT_TYPES[selectedType as keyof typeof DOCUMENT_TYPES]?.label} here
              </p>
              <p className="text-sm text-gray-500 mb-4">or click to browse</p>
              <p className="text-xs text-gray-400">Supports: PDF, JPG, PNG (max 10MB)</p>
            </>
          )}
        </div>

        {isUploading && (
          <div className="mt-4 p-4 bg-blue-50 border border-blue-200 rounded-lg">
            <div className="flex items-center gap-3">
              <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-indigo-600"></div>
              <p className="text-sm text-blue-700">Uploading and processing with AI...</p>
            </div>
          </div>
        )}
      </div>

      {/* Documents List */}
      <div className="space-y-4">
        <h2 className="text-xl font-semibold text-gray-900">Your Documents</h2>

        {documents.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-xl">
            <FileText className="w-16 h-16 text-gray-300 mx-auto mb-4" />
            <p className="text-gray-600">No documents uploaded yet</p>
            <p className="text-sm text-gray-500 mt-1">Upload your first document to get started</p>
          </div>
        ) : (
          <div className="grid gap-4">
            {documents.map((doc) => {
              const statusConfig = VERIFICATION_STATUS[doc.verificationStatus as keyof typeof VERIFICATION_STATUS];
              const StatusIcon = statusConfig?.icon || AlertCircle;

              return (
                <div
                  key={doc.documentId}
                  className="bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-shadow"
                >
                  <div className="flex items-start gap-4">
                    <div className="text-4xl">
                      {DOCUMENT_TYPES[doc.documentType as keyof typeof DOCUMENT_TYPES]?.icon || '📄'}
                    </div>

                    <div className="flex-1 min-w-0">
                      <div className="flex items-start justify-between mb-2">
                        <div>
                          <h3 className="font-semibold text-gray-900">{doc.fileName}</h3>
                          <p className="text-sm text-gray-600">
                            {DOCUMENT_TYPES[doc.documentType as keyof typeof DOCUMENT_TYPES]?.label || doc.documentType}
                          </p>
                        </div>

                        <span className={`px-3 py-1 rounded-full text-sm font-medium flex items-center gap-1 ${statusConfig?.bgColor} ${statusConfig?.color}`}>
                          <StatusIcon className="w-4 h-4" />
                          {statusConfig?.label}
                        </span>
                      </div>

                      <div className="flex items-center gap-4 text-sm text-gray-500 mb-3">
                        <span>{formatFileSize(doc.fileSizeBytes)}</span>
                        <span>•</span>
                        <span>{new Date(doc.uploadedAt).toLocaleDateString()}</span>
                        {doc.ocrConfidence !== null && (
                          <>
                            <span>•</span>
                            <span className="flex items-center gap-1">
                              AI Confidence: 
                              <span className={`font-semibold ${
                                doc.ocrConfidence >= 0.85 ? 'text-green-600' :
                                doc.ocrConfidence >= 0.70 ? 'text-yellow-600' : 'text-orange-600'
                              }`}>
                                {(doc.ocrConfidence * 100).toFixed(0)}%
                              </span>
                            </span>
                          </>
                        )}
                      </div>

                      {/* OCR Extracted Data Preview */}
                      {doc.ocrExtractedData && (
                        <div className="bg-gradient-to-br from-indigo-50 to-purple-50 p-4 rounded-lg mb-3 border border-indigo-100">
                          <p className="text-sm font-medium text-indigo-900 mb-2">✨ AI Extracted Information:</p>
                          <div className="grid grid-cols-2 gap-3 text-sm">
                            {Object.entries(doc.ocrExtractedData).map(([key, value]) => (
                              <div key={key}>
                                <span className="text-gray-600 capitalize">{key.replace(/_/g, ' ')}:</span>
                                <span className="ml-2 font-medium text-gray-900">{String(value)}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {doc.verificationNotes && (
                        <div className="bg-yellow-50 border border-yellow-200 p-3 rounded-lg mb-3">
                          <p className="text-sm text-yellow-800">
                            <strong>Note:</strong> {doc.verificationNotes}
                          </p>
                        </div>
                      )}

                      <div className="flex gap-2">
                        <a
                          href={doc.fileUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="px-4 py-2 bg-indigo-50 text-indigo-700 rounded-lg hover:bg-indigo-100 transition-colors text-sm font-medium flex items-center gap-2"
                        >
                          <Eye className="w-4 h-4" />
                          View
                        </a>
                        <a
                          href={doc.fileUrl}
                          download
                          className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors text-sm font-medium flex items-center gap-2"
                        >
                          <Download className="w-4 h-4" />
                          Download
                        </a>
                        <button
                          onClick={() => deleteDocument(doc.documentId)}
                          className="px-4 py-2 bg-red-50 text-red-700 rounded-lg hover:bg-red-100 transition-colors text-sm font-medium flex items-center gap-2 ml-auto"
                        >
                          <Trash2 className="w-4 h-4" />
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
