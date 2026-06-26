import React from 'react';
import { ApolloProvider } from '@apollo/client';
import { client } from './apollo';
import AIRebalancingDashboard from './components/AIRebalancingDashboard';
import AISimulationDashboard from './components/AISimulationDashboard';
import RiskAlphaDashboard from './components/RiskAlphaDashboard';
import AttributionAlphaDashboard from './components/AttributionAlphaDashboard';
import { BrowserRouter as Router, Routes, Route, Link, NavLink } from 'react-router-dom';

function App() {
  return (
    <ApolloProvider client={client}>
      <Router>
        <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 text-white">
          <nav className="bg-slate-900/50 border-b border-slate-700 px-8 py-4 flex items-center justify-between">
            <Link to="/" className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-purple-500 bg-clip-text text-transparent">
              AI Portfolio Alpha
            </Link>
            <div className="flex items-center gap-6">
              <NavLink to="/" className={({ isActive }) => isActive ? "text-blue-400 font-bold" : "text-slate-300 hover:text-white"}>Rebalancing</NavLink>
              <NavLink to="/simulate" className={({ isActive }) => isActive ? "text-blue-400 font-bold" : "text-slate-300 hover:text-white"}>Simulator</NavLink>
              <NavLink to="/risk" className={({ isActive }) => isActive ? "text-blue-400 font-bold" : "text-slate-300 hover:text-white"}>Risk</NavLink>
              <NavLink to="/attribution" className={({ isActive }) => isActive ? "text-blue-400 font-bold" : "text-slate-300 hover:text-white"}>Attribution</NavLink>
            </div>
          </nav>
          <main className="p-8">
            <Routes>
              <Route path="/" element={<AIRebalancingDashboard />} />
              <Route path="/simulate" element={<AISimulationDashboard />} />
              <Route path="/risk" element={<RiskAlphaDashboard />} />
              <Route path="/attribution" element={<AttributionAlphaDashboard />} />
            </Routes>
          </main>
        </div>
      </Router>
    </ApolloProvider>
  );
}

export default App;