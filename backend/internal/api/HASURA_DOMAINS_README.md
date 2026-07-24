This file contains an example Hasura metadata snippet and instructions for mapping the `data_domain` table into Hasura.

1) Apply the SQL migration `004_create_data_domains_table.up.sql` to your database.

2) In Hasura Console -> Data -> Track tables, track the `public.data_domain` table.

3) Add permissions for roles you need (e.g., admin, user) for select/insert/update/delete.

4) If you want GraphQL queries/mutations names or custom root fields, use Hasura's metadata editor. The file `hasura_data_domains_metadata_example.json` is a minimal example; export your full metadata after making changes.

5) Consider adding event triggers if you want to propagate domain changes to other systems.
