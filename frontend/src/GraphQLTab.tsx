// React import removed (not needed with the new JSX transform)

export default function GraphQLTab({ graphql }: { graphql?: string }) {
  return (
    <div className="graphql-tab">
      <button onClick={() => navigator.clipboard.writeText(graphql || '')}>Copy GraphQL</button>
      <a href={`/graphiql?query=${encodeURIComponent(graphql || '')}`} target="_blank" rel="noreferrer">Open in GraphiQL</a>
      <pre>{graphql}</pre>
    </div>
  );
}