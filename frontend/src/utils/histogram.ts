// utils/histogram.ts
export type Bin = { x0: number; x1: number; count: number };

export function computeHistogram(samples: number[], bins = 40): Bin[] {
  if (!samples?.length) return [];
  const min = Math.min(...samples);
  const max = Math.max(...samples);
  const width = (max - min) / bins || 1;
  const edges = Array.from({ length: bins + 1 }, (_, i) => min + i * width);
  const counts = Array(bins).fill(0);
  for (const v of samples) {
    let j = Math.floor((v - min) / width);
    if (j < 0) j = 0;
    if (j >= bins) j = bins - 1;
    counts[j]++;
  }
  return counts.map((c, i) => ({ x0: edges[i], x1: edges[i + 1], count: c }));
}

export function formatBinLabel(b: Bin, digits = 3) {
  return `${b.x0.toFixed(digits)}–${b.x1.toFixed(digits)}`;
}
