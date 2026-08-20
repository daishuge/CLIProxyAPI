import { describe, expect, test } from 'bun:test';
import { SPONSORS } from '../src/features/config/sponsors';
import {
  TEMPORARILY_HIDDEN_SPONSOR_BRANDS,
  isTemporarilyHiddenSponsorBrand,
} from '../src/features/providers/sponsorDefinitions';

const SPONSOR_BRANDS = [
  'apikeyFun',
  'code0',
  'fennoAI',
  'qiniuCloud',
  'lmuAI',
  'infistar',
  'kimi',
] as const;

describe('PPAP sponsor-free UI', () => {
  test('hides affiliate links and every sponsor-branded provider entry', () => {
    expect(SPONSORS).toEqual([]);
    expect(TEMPORARILY_HIDDEN_SPONSOR_BRANDS.size).toBe(SPONSOR_BRANDS.length);
    for (const brand of SPONSOR_BRANDS) {
      expect(isTemporarilyHiddenSponsorBrand(brand)).toBe(true);
    }
  });
});
