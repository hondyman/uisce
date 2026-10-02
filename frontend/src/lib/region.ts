// The region the user selected, or '' when none is. There is deliberately no default:
// an unselected region is sent as nothing and the backend rejects the request
// ("region is required"), rather than silently scoping it to a guessed region.
export function getSelectedRegion(): string {
  return localStorage.getItem('selected_region') || '';
}

export function setSelectedRegion(region: string): void {
  localStorage.setItem('selected_region', region);
}
