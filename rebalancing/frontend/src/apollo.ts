import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { getMainDefinition } from '@apollo/client/utilities';
import { createClient } from 'graphql-ws';

const httpLink = new HttpLink({
  uri: 'http://localhost:8080/v1/graphql',
  headers: {
    'x-hasura-admin-secret': 'secret' // or JWT token
  }
});

const wsLink = new GraphQLWsLink(
  createClient({
    url: 'ws://localhost:8080/v1/graphql',
    connectionParams: {
      headers: {
        'x-hasura-admin-secret': 'secret'
      }
    }
  })
);

const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    );
  },
  wsLink,
  httpLink
);

export const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache()
});
