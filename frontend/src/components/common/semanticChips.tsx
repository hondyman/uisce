import { CORE_COLOR, CUSTOM_COLOR, renderCoreCustomIcons } from './CoreCustomIcons';

export { CORE_COLOR, CUSTOM_COLOR };

export function renderCoreCustomChips(option?: unknown) {
  return renderCoreCustomIcons({ option, variant: 'chip-icon' });
}

export default renderCoreCustomChips;