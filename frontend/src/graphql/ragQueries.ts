import { gql } from '@apollo/client';

export const SEARCH_RAG = gql`
  mutation SearchRAG($query: String!, $limit: Int!) {
    searchRAG(input: { query: $query, limit: $limit }) {
      results {
        chunk_id
        document_id
        content
        score
        metadata
      }
    }
  }
`;

export const UPLOAD_DOCUMENT = gql`
  mutation UploadDocument($filePath: String!, $title: String!) {
    uploadDocument(input: { file_path: $filePath, title: $title }) {
      document_id
      status
    }
  }
`;
