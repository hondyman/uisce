/**
 * Color Palette Management for Node and Edge Types
 * Ensures colors are unique across types and provides color management utilities
 */

export interface ColorPalette {
  id: string;
  name: string;
  hex: string;
  isCustom: boolean;
  lastUsedBy?: string; // node_type_id or edge_type_id
}

// Default color palette - ensures variety and accessibility
export const DEFAULT_COLORS: ColorPalette[] = [
  { id: 'blue-500', name: 'Blue', hex: '#3B82F6', isCustom: false },
  { id: 'purple-500', name: 'Purple', hex: '#A855F7', isCustom: false },
  { id: 'green-500', name: 'Green', hex: '#10B981', isCustom: false },
  { id: 'red-500', name: 'Red', hex: '#EF4444', isCustom: false },
  { id: 'orange-500', name: 'Orange', hex: '#F97316', isCustom: false },
  { id: 'pink-500', name: 'Pink', hex: '#EC4899', isCustom: false },
  { id: 'cyan-500', name: 'Cyan', hex: '#06B6D4', isCustom: false },
  { id: 'amber-500', name: 'Amber', hex: '#FBBF24', isCustom: false },
  { id: 'teal-500', name: 'Teal', hex: '#14B8A6', isCustom: false },
  { id: 'indigo-500', name: 'Indigo', hex: '#6366F1', isCustom: false },
  { id: 'rose-500', name: 'Rose', hex: '#F43F5E', isCustom: false },
  { id: 'lime-500', name: 'Lime', hex: '#84CC16', isCustom: false },
];

/**
 * Validates if a color string is a valid hex color
 */
export function isValidHexColor(color: string): boolean {
  return /^#[0-9A-F]{6}$/i.test(color);
}

/**
 * Converts hex color to RGB
 */
export function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16),
  } : null;
}

/**
 * Calculates perceived brightness of a color (0-255)
 * Uses luminance formula from WCAG
 */
export function getColorBrightness(hex: string): number {
  const rgb = hexToRgb(hex);
  if (!rgb) return 128;
  
  const { r, g, b } = rgb;
  return (r * 299 + g * 587 + b * 114) / 1000;
}

/**
 * Calculates the color distance (0-441) between two hex colors
 * Based on Euclidean distance in RGB space
 */
export function getColorDistance(hex1: string, hex2: string): number {
  const rgb1 = hexToRgb(hex1);
  const rgb2 = hexToRgb(hex2);
  
  if (!rgb1 || !rgb2) return 441;
  
  const rDiff = rgb1.r - rgb2.r;
  const gDiff = rgb1.g - rgb2.g;
  const bDiff = rgb1.b - rgb2.b;
  
  return Math.sqrt(rDiff * rDiff + gDiff * gDiff + bDiff * bDiff);
}

/**
 * Finds the best available color from palette that doesn't conflict with used colors
 * Minimum distance threshold: 100 (on scale of 0-441)
 */
export function findNonConflictingColor(
  usedColors: string[],
  palette: ColorPalette[] = DEFAULT_COLORS,
  minDistance: number = 100
): string | null {
  for (const color of palette) {
    const isConflicting = usedColors.some(
      used => getColorDistance(color.hex, used) < minDistance
    );
    
    if (!isConflicting) {
      return color.hex;
    }
  }
  
  // If no color available in palette, find the least similar one
  if (usedColors.length > 0 && palette.length > 0) {
    let bestColor = palette[0].hex;
    let maxDistance = 0;
    
    for (const color of palette) {
      const minDistToUsed = Math.min(
        ...usedColors.map(used => getColorDistance(color.hex, used))
      );
      
      if (minDistToUsed > maxDistance) {
        maxDistance = minDistToUsed;
        bestColor = color.hex;
      }
    }
    
    return bestColor;
  }
  
  return palette.length > 0 ? palette[0].hex : null;
}

/**
 * Gets a contrast-friendly text color (black or white) based on background color
 */
export function getContrastColor(bgHex: string): string {
  const brightness = getColorBrightness(bgHex);
  return brightness > 128 ? '#000000' : '#FFFFFF';
}

/**
 * Suggests a next color based on recently used colors
 */
export function suggestNextColor(
  usedColors: string[],
  palette: ColorPalette[] = DEFAULT_COLORS
): string {
  const available = findNonConflictingColor(usedColors, palette);
  return available || palette[Math.floor(Math.random() * palette.length)].hex;
}
