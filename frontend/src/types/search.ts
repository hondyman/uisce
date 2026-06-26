export interface SearchResult<T = any> {
  id: string | number;
  text: string;
  subtext?: string;
  // optional payload to carry full object (e.g., NodeType)
  payload?: T;
}

export type SearchResultMap<T = any> = Array<SearchResult<T>>;
