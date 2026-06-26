import React, { createContext, useContext, useState, ReactNode } from 'react';

type GlobalSearchContextType = {
  searchTerm: string;
  setSearchTerm: (s: string) => void;
};

const defaultValue: GlobalSearchContextType = {
  searchTerm: '',
  setSearchTerm: () => {},
};

export const GlobalSearchContext = createContext<GlobalSearchContextType>(defaultValue);

export const GlobalSearchProvider: React.FC<{
  children: ReactNode;
  value?: Partial<GlobalSearchContextType>;
}> = ({ children, value }) => {
  // allow an external controlled value or fall back to internal state
  const [internalSearchTerm, setInternalSearchTerm] = useState<string>(value?.searchTerm ?? '');

  const searchTerm = value?.searchTerm ?? internalSearchTerm;
  const setSearchTerm = value?.setSearchTerm ?? setInternalSearchTerm;

  return (
    <GlobalSearchContext.Provider value={{ searchTerm, setSearchTerm }}>
      {children}
    </GlobalSearchContext.Provider>
  );
};

export const useGlobalSearch = () => useContext(GlobalSearchContext);
