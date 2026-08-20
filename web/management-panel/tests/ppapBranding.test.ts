import { describe, expect, test } from 'bun:test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import en from '../src/i18n/locales/en.json';
import ru from '../src/i18n/locales/ru.json';
import zhCN from '../src/i18n/locales/zh-CN.json';
import zhTW from '../src/i18n/locales/zh-TW.json';

const root = join(import.meta.dir, '..');

describe('PPAP branding', () => {
  test('uses PPAP identity in every locale', () => {
    for (const locale of [en, ru, zhCN, zhTW]) {
      expect(locale.title.abbr).toBe('PPAP');
      expect(locale.title.login).toContain('Playful Proxy API');
      expect(locale.sidebar.subtitle).toContain('Playful Proxy API');
    }
  });

  test('uses PPAP title, logo, and repository links', () => {
    expect(readFileSync(join(root, 'index.html'), 'utf8')).toContain(
      '<title>Playful Proxy API Panel</title>'
    );
    expect(readFileSync(join(root, 'src/assets/logoInline.ts'), 'utf8')).toContain('>PPAP</text>');
    const systemPage = readFileSync(join(root, 'src/pages/SystemPage.tsx'), 'utf8');
    expect(systemPage.match(/github\.com\/daishuge\/playful-proxy-api-panel/g)?.length).toBe(2);
  });
});
