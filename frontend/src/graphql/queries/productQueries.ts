import { gql } from '@apollo/client';

export const GET_ALL_PRODUCTS = gql`
  query GetAllProducts {
    alpha_product {
      id
      product_name
    }
  }
`;
