# Backend Setup

This directory contains the backend setup for the Workday Benefits application.

## Services

The backend consists of two services, defined in `docker-compose.yml`:

- `postgres`: A PostgreSQL database to store the application data.
- `graphql-engine`: A Hasura GraphQL engine to provide a GraphQL API over the database.

## Running the Services

To run the services, navigate to this directory and run:

```
docker-compose up -d
```

## Security

The `docker-compose.yml` file contains a hardcoded admin secret for the Hasura GraphQL engine. This is for development purposes only and should be replaced with a more secure method for managing secrets in a production environment.