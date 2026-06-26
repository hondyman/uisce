import ConversationalQueryInterface from '../ConversationalQueryInterface';
import './ConversationalQueryPage.css';

export default function ConversationalQueryPage() {
  // Mock user and tenant data - in a real app, this would come from authentication context
  const currentUser = 'analyst123';
  const currentTenant = 'acme_corp';
  const currentDatasource = 'orders_db';

  const handleQueryGenerated = (_query: any) => {
    // Here you could:
    // - Save the query to favorites
    // - Execute the query immediately
    // - Add it to a query history
    // - Navigate to a results page
  };

  return (
    <div className="conversational-query-page">
      <div className="page-header">
        <h1>Conversational Query Builder</h1>
        <p>
          Describe the data you need in natural language, and I'll help you create a compliant query
          through an interactive conversation. I'll clarify ambiguities, suggest improvements, and
          ensure compliance with your organization's policies.
        </p>
      </div>

      <div className="page-content">
        <ConversationalQueryInterface
          currentDatasource={currentDatasource}
          currentUser={currentUser}
          currentTenant={currentTenant}
          onQueryGenerated={handleQueryGenerated}
        />
      </div>
    </div>
  );
}
