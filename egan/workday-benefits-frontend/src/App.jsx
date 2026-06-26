import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ApolloClient, InMemoryCache, ApolloProvider } from '@apollo/client';
import Layout from './components/Layout';
import ValidationRulesHeader from './components/ValidationRulesHeader';
import SearchAndFilters from './components/SearchAndFilters';
import BenefitsSummaryPage from './pages/BenefitsSummaryPage';
import PlanDetailsPage from './pages/PlanDetailsPage';
import ReviewAndSubmitPage from './pages/ReviewAndSubmitPage';

const client = new ApolloClient({
  uri: '/graphql', // This will be intercepted by msw
  cache: new InMemoryCache(),
});

const ValidationRulesPage = () => (
  <Layout>
    <ValidationRulesHeader />
    <SearchAndFilters />
  </Layout>
);

function App() {
  return (
    <ApolloProvider client={client}>
      <Router>
        <Routes>
          <Route path="/" element={<ValidationRulesPage />} />
          <Route path="/benefits-summary" element={<BenefitsSummaryPage />} />
          <Route path="/plan-details" element={<PlanDetailsPage />} />
          <Route path="/review-submit" element={<ReviewAndSubmitPage />} />
        </Routes>
      </Router>
    </ApolloProvider>
  );
}

export default App;
