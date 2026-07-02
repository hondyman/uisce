import { i18n } from '../../locale/locale';
/**
 * transform message to locale language
 * @param message error msg
 * @param locale language setting
 */
function transform(message, locale) {
    const regex = /{([^}]+)}/g;
    return message.replace(regex, (_, key) => i18n[locale][key] || '');
}
export { transform };
