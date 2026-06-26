import { gql, useSubscription } from '@apollo/client'

const SUB = gql`
  subscription { temporal_workflows { workflow_id status } }
`

export default function ExecutionMonitor() {
  const { data, error } = useSubscription(SUB, {
    onError: (err) => console.error('Subscription error:', err),
  })

  if (error) {
    return <div style={{ color: 'red' }}>Live update failed: {error.message}</div>
  }

  return (
    <div>
      {data?.temporal_workflows?.map((w: any) => (
        <div key={w.workflow_id}>{w.status}</div>
      ))}
    </div>
  )
}
