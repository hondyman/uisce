/**
 * Utility functions for IP address validation and pattern matching
 */

// Parse IPv4 dotted string into 32-bit number
const ipv4ToInt = (ip: string): number | null => {
  const parts = ip.split('.');
  if (parts.length !== 4) return null;
  let n = 0;
  for (const p of parts) {
    const v = Number(p);
    if (!Number.isInteger(v) || v < 0 || v > 255) return null;
    n = (n << 8) | v;
  }
  // Use >>> 0 to force unsigned 32-bit
  return n >>> 0;
};

export const isValidIPAddress = (ip: string): boolean => {
  const trimmed = ip.trim();

  // CIDR: a.b.c.d/nn where 0 <= nn <= 32 and base IPv4
  if (trimmed.includes('/')) {
    const [base, maskStr] = trimmed.split('/');
    const mask = Number(maskStr);
    if (!Number.isInteger(mask) || mask < 0 || mask > 32) return false;
    const baseInt = ipv4ToInt(base);
    return baseInt !== null;
  }

  // Wildcards or plain IPv4: each octet either * or 0-255
  const ipv4Pattern = /^(\d{1,3}|\*)\.(\d{1,3}|\*)\.(\d{1,3}|\*)\.(\d{1,3}|\*)$/;
  if (!ipv4Pattern.test(trimmed)) return false;

  const parts = trimmed.split('.');
  return parts.every(part => {
    if (part === '*') return true;
    const num = parseInt(part, 10);
    return num >= 0 && num <= 255;
  });
};

export const normalizeIPPattern = (ip: string): string => {
  return ip.trim().toLowerCase();
};

export const ipPatternsOverlap = (pattern1: string, pattern2: string): boolean => {
  const toRange = (p: string): [number, number] | null => {
    const s = normalizeIPPattern(p);
    // CIDR
    if (s.includes('/')) {
      const [base, maskStr] = s.split('/');
      const mask = Number(maskStr);
      const baseInt = ipv4ToInt(base);
      if (baseInt === null || !Number.isInteger(mask) || mask < 0 || mask > 32) return null;
      const hostBits = 32 - mask;
      const maskInt = mask === 0 ? 0 : (~((1 << hostBits) - 1)) >>> 0;
      const network = (baseInt & maskInt) >>> 0;
      const broadcast = (network | (~maskInt >>> 0)) >>> 0;
      return [network >>> 0, broadcast >>> 0];
    }
    // Wildcards / plain IPv4
    const parts = s.split('.');
    if (parts.length !== 4) return null;
    const minParts: number[] = [];
    const maxParts: number[] = [];
    for (const part of parts) {
      if (part === '*') {
        minParts.push(0); maxParts.push(255);
      } else {
        const v = Number(part);
        if (!Number.isInteger(v) || v < 0 || v > 255) return null;
        minParts.push(v); maxParts.push(v);
      }
    }
    const min = ((minParts[0] << 24) | (minParts[1] << 16) | (minParts[2] << 8) | minParts[3]) >>> 0;
    const max = ((maxParts[0] << 24) | (maxParts[1] << 16) | (maxParts[2] << 8) | maxParts[3]) >>> 0;
    return [min, max];
  };

  const r1 = toRange(pattern1);
  const r2 = toRange(pattern2);
  if (!r1 || !r2) return false;
  const [a1, a2] = r1;
  const [b1, b2] = r2;
  return !(a2 < b1 || b2 < a1);
};

export const expandIPPattern = (pattern: string): string[] => {
  const parts = pattern.split('.');
  
  if (parts.length !== 4) {
    return [pattern];
  }

  const expanded: string[][] = [];
  
  for (let i = 0; i < 4; i++) {
    if (parts[i] === '*') {
      expanded[i] = Array.from({ length: 256 }, (_, index) => index.toString());
    } else {
      expanded[i] = [parts[i]];
    }
  }

  // Generate cartesian product (limited to prevent memory issues)
  const maxExpanded = 1000; // Limit expansion for performance
  const result: string[] = [];
  
  function cartesian(arrays: string[][], current: string[] = [], index: number = 0) {
    if (index === arrays.length) {
      result.push(current.join('.'));
      return;
    }
    
    if (result.length >= maxExpanded) {
      return;
    }
    
    for (const item of arrays[index]) {
      cartesian(arrays, [...current, item], index + 1);
      if (result.length >= maxExpanded) {
        break;
      }
    }
  }
  
  cartesian(expanded);
  return result;
};

export const getIPPatternDescription = (pattern: string): string => {
  const s = pattern.trim();
  if (s.includes('/')) {
    const [base, maskStr] = s.split('/');
    const mask = Number(maskStr);
    if (!Number.isInteger(mask) || mask < 0 || mask > 32) return s;
    const count = mask === 32 ? 1 : Math.pow(2, 32 - mask);
    return `CIDR range: ${base}/${mask} (${count.toLocaleString()} addresses)`;
  }

  const parts = s.split('.');
  if (parts.length !== 4) return s;
  const wildcardCount = parts.filter(part => part === '*').length;
  if (wildcardCount === 0) {
    return `Single IP address: ${s}`;
  } else if (wildcardCount === 1) {
    const wildcardIndex = parts.indexOf('*');
    const networkParts = parts.slice(0, wildcardIndex);
    return `Network range: ${networkParts.join('.')}.* (256 addresses)`;
  } else if (wildcardCount === 2) {
    return `Large network range (65,536 addresses)`;
  } else if (wildcardCount === 3) {
    return `Very large network range (16,777,216 addresses)`;
  } else {
    return `All IP addresses (4,294,967,296 addresses)`;
  }
};

export const suggestSimilarPatterns = (pattern: string): string[] => {
  const parts = pattern.split('.');
  
  if (parts.length !== 4) {
    return [];
  }

  const suggestions: string[] = [];
  
  // Suggest broader patterns (more wildcards)
  for (let i = 3; i >= 0; i--) {
    const broaderParts = [...parts];
    broaderParts[i] = '*';
    suggestions.push(broaderParts.join('.'));
  }
  
  // Suggest narrower patterns (less wildcards)
  for (let i = 0; i < 4; i++) {
    if (parts[i] === '*') {
      const narrowerParts = [...parts];
      narrowerParts[i] = '1'; // Example specific value
      suggestions.push(narrowerParts.join('.'));
    }
  }
  
  return suggestions.filter((s, index, arr) => 
    arr.indexOf(s) === index && s !== pattern
  );
};

export const formatIPForDisplay = (ip: string): string => {
  // Add visual indicators for patterns
  if (ip.includes('/')) return `${ip} (CIDR)`;
  if (ip.includes('*')) return `${ip} (pattern)`;
  return ip;
};

export const sortIPAddresses = (ips: string[]): string[] => {
  return ips.sort((a, b) => {
    const partsA = a.split('.').map(part => part === '*' ? -1 : parseInt(part, 10));
    const partsB = b.split('.').map(part => part === '*' ? -1 : parseInt(part, 10));
    
    for (let i = 0; i < 4; i++) {
      if (partsA[i] !== partsB[i]) {
        return partsA[i] - partsB[i];
      }
    }
    
    return 0;
  });
};
