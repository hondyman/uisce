import React from 'react';

const Pagination = () => {
  return (
    <div className="flex items-center justify-between p-4 border-t border-gray-200 dark:border-gray-700">
      <p className="text-sm text-gray-500 dark:text-gray-400">Showing <span className="font-medium">1</span> to <span className="font-medium">5</span> of <span className="font-medium">20</span> results</p>
      <div className="inline-flex items-center -space-x-px">
        <button className="px-3 py-2 text-sm font-medium leading-5 text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-l-lg hover:bg-gray-100 dark:hover:bg-gray-800">Previous</button>
        <button className="px-3 py-2 text-sm font-medium leading-5 text-white bg-primary border border-primary">1</button>
        <button className="px-3 py-2 text-sm font-medium leading-5 text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-800">2</button>
        <button className="px-3 py-2 text-sm font-medium leading-5 text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 hover:bg-gray-100 dark:hover:bg-gray-800">3</button>
        <button className="px-3 py-2 text-sm font-medium leading-5 text-gray-500 dark:text-gray-400 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-r-lg hover:bg-gray-100 dark:hover:bg-gray-800">Next</button>
      </div>
    </div>
  );
};

export default Pagination;