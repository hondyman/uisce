export function getSelectedRegion(): string {
  const stored = localStorage.getItem('selected_region');
  // Always return a region - fallback to us-west if not set
  return stored || 'us-west';
}

export function setSelectedRegion(region: string): void {
  localStorage.setItem('selected_region', region);
}
