import { LOCALE_TYPE } from './types';
/**
 * transform message to locale language
 * @param message error msg
 * @param locale language setting
 */
declare function transform(message: string, locale: LOCALE_TYPE): string;
export { transform };
