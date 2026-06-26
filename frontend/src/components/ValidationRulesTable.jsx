
import { useQuery, gql } from '@apollo/client';

const GET_VALIDATION_RULES = gql`
  query GetValidationRules {
    validation_rules {
      id
      name
      status
      benefitPlan: scope_ref
      triggerEvent: scope_type
    }
  }
`;

const StatusBadge = ({ status }) => {
  const baseClasses = "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium";
  const statusClasses = {
    Active: "bg-green-100 dark:bg-green-900/50 text-green-800 dark:text-green-300",
    Inactive: "bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-300",
    Draft: "bg-yellow-100 dark:bg-yellow-900/50 text-yellow-800 dark:text-yellow-300",
  };
  const dotClasses = {
    Active: "text-green-500",
    Inactive: "text-gray-500",
    Draft: "text-yellow-500",
  }

  return (
    <span className={`${baseClasses} ${statusClasses[status]}`}>
      <svg className={`mr-1.5 h-2 w-2 ${dotClasses[status]}`} fill="currentColor" viewBox="0 0 8 8">
        <circle cx="4" cy="4" r="3"></circle>
      </svg>
      {status}
    </span>
  );
};

const ValidationRulesTable = () => {
  const { loading, error, data } = useQuery(GET_VALIDATION_RULES);

  if (loading) return <p>Loading...</p>;
  if (error) return <p>Error :(</p>;

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left">
        <thead>
          <tr className="bg-gray-50 dark:bg-gray-800/50">
            <th className="p-4 text-xs font-medium uppercase text-gray-500 dark:text-gray-400 w-2/5">Rule Name</th>
            <th className="p-4 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Status</th>
            <th className="p-4 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Benefit Plan</th>
            <th className="p-4 text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Trigger Event</th>
            <th className="p-4 text-xs font-medium uppercase text-gray-500 dark:text-gray-400 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
          {data.validation_rules.map((rule) => (
            <tr key={rule.id}>
              <td className="p-4 text-sm font-medium text-gray-900 dark:text-white">{rule.name}</td>
              <td className="p-4 text-sm">
                <StatusBadge status={rule.status} />
              </td>
              <td className="p-4 text-sm text-gray-500 dark:text-gray-400">{rule.benefitPlan}</td>
              <td className="p-4 text-sm text-gray-500 dark:text-gray-400">{rule.triggerEvent}</td>
              <td className="p-4 text-sm font-medium text-right">
                <button className="text-primary hover:underline">Edit</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default ValidationRulesTable;
