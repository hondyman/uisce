
import ValidationRulesTable from '../components/ValidationRulesTable';
import Pagination from '../components/Pagination';
import SVGIcon from './relationship/SVGIcon';

const SearchAndFilters = () => {
  return (
    <div className="flex flex-col bg-white dark:bg-gray-900 border border-gray-200 dark:border-gray-700 rounded-xl">
      <div className="flex flex-wrap items-center gap-4 p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex-1 min-w-[250px]">
          <label className="flex flex-col h-10 w-full">
            <div className="flex w-full flex-1 items-stretch rounded-lg h-full bg-gray-100 dark:bg-gray-800">
                <div className="text-gray-500 dark:text-gray-400 flex items-center justify-center pl-3">
                <SVGIcon name="search" ariaLabel="search" />
              </div>
              <input className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden text-gray-900 dark:text-white focus:outline-0 focus:ring-0 border-none bg-transparent h-full placeholder:text-gray-500 dark:placeholder:text-gray-400 px-2 text-sm font-normal" placeholder="Search for a rule by name..." value=""/>
            </div>
          </label>
        </div>
        <div className="flex gap-2">
            <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-gray-100 dark:bg-gray-800 px-4 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700">
            <p className="text-sm font-medium">Status: All</p>
            <SVGIcon name="arrow_drop_down" ariaLabel="expand" />
          </button>
          <button className="flex h-10 shrink-0 items-center justify-center gap-x-2 rounded-lg bg-gray-100 dark:bg-gray-800 px-4 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700">
            <p className="text-sm font-medium">Benefit Plan: All</p>
            <SVGIcon name="arrow_drop_down" ariaLabel="expand" />
          </button>
        </div>
      </div>
      <ValidationRulesTable />
      <Pagination />
    </div>
  );
};

export default SearchAndFilters;
