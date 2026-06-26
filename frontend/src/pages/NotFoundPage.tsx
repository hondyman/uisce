// React import not required with the new JSX transform
import { Link } from 'react-router-dom';

const NotFoundPage: React.FC = () => {
  return (
    <div>
      <h1>404 - Not Found</h1>
      <p>Sorry, the page you are looking for does not exist.</p>
      <Link to="/">Go back to Home</Link>
    </div>
  );
};

export default NotFoundPage;