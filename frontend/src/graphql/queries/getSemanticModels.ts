import { gql } from '@apollo/client';

export const GET_SEMANTIC_MODELS = gql`
  query GetSemanticModels {
    semantic_models {
      id
      name
      model_type
      dataset_id
      configuration
      created_at
      updated_at
    }
  }
`;
