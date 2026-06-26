import React from 'react';
import { Link } from 'react-router-dom';

const Layout = ({ children }) => {
  return (
    <div className="relative flex min-h-screen w-full flex-row">
      {/* SideNavBar */}
      <aside className="flex h-screen w-64 flex-col border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 sticky top-0">
        <div className="flex h-16 shrink-0 items-center gap-2 border-b border-gray-200 dark:border-gray-700 px-6">
          <span className="material-symbols-outlined text-primary text-3xl">shield_with_heart</span>
          <h2 className="text-lg font-bold text-gray-900 dark:text-white">Benefits Admin</h2>
        </div>
        <nav className="flex flex-col justify-between grow p-4">
          <div className="flex flex-col gap-2">
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="#">
              <span className="material-symbols-outlined">dashboard</span>
              <p className="text-sm font-medium">Dashboard</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg bg-primary/10 px-3 py-2 text-primary dark:text-primary" to="/">
              <span className="material-symbols-outlined fill">gavel</span>
              <p className="text-sm font-medium">Validation Rules</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="/benefits-summary">
              <span className="material-symbols-outlined">person</span>
              <p className="text-sm font-medium">Benefits Summary</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="/plan-details">
              <span className="material-symbols-outlined">description</span>
              <p className="text-sm font-medium">Plan Details</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="/review-submit">
              <span className="material-symbols-outlined">task_alt</span>
              <p className="text-sm font-medium">Review & Submit</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="#">
              <span className="material-symbols-outlined">history</span>
              <p className="text-sm font-medium">Logs/History</p>
            </Link>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="#">
              <span className="material-symbols-outlined">settings</span>
              <p className="text-sm font-medium">Settings</p>
            </Link>
          </div>
          <div className="flex flex-col">
            <div className="flex items-center gap-4 p-2 border-t border-gray-200 dark:border-gray-700 pt-4 mt-2">
              <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuDTWws48ehInyfnPNmXf02WfUJWla1eV9kPcn1QxP020rkDZm8bzfEpTYzUL5cZVhE2TMqCmjZyAXM0r7FZybknhIRdAghueBlrjUCtXqo7gXmaC9UyunATA6FR2UFpNSojRDDoqzSIu_WP6PPsq3vkH4_kXsZpoiYWfegMVxx3kVgZwoQh0VqtnSRE7-NSX4xXDwqlUEqhu2hqbMhcPjOU5Y_hr1em5Nttc50fCusaDR4rGQuFUPQweJbJn3PxQTlcX42w41Su1PJp")'}}></div>
              <div className="flex flex-col">
                <h1 className="text-gray-900 dark:text-white text-sm font-medium leading-normal">Admin User</h1>
                <p className="text-gray-500 dark:text-gray-400 text-xs font-normal leading-normal">admin.user@workday.com</p>
              </div>
            </div>
            <Link className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800" to="#">
              <span className="material-symbols-outlined">logout</span>
              <p className="text-sm font-medium">Logout</p>
            </Link>
          </div>
        </nav>
      </aside>
      {/* Main Content */}
      <main className="flex-1 flex flex-col p-8 bg-background-light dark:bg-background-dark">
        {children}
      </main>
    </div>
  );
};

export default Layout;