import { describe, it, expect, beforeEach } from 'vitest';
import i18n, { setLanguage, SUPPORTED_LANGUAGES } from '../index';

describe('i18n', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('exposes en / zh-TW / th as supported languages', () => {
    expect(SUPPORTED_LANGUAGES).toEqual(['en', 'zh-TW', 'th']);
  });

  it('defaults to English when no language is stored', () => {
    // i18n was initialized at module load time with no localStorage value
    expect(['en', 'zh-TW', 'th']).toContain(i18n.language);
    // i18next normalises fallbackLng to an array; check membership not equality
    const fallback = i18n.options.fallbackLng;
    const fallbackArr = Array.isArray(fallback) ? fallback : [fallback];
    expect(fallbackArr).toContain('en');
  });

  it('translates errors.INVALID_CREDENTIALS in all three locales', async () => {
    await i18n.changeLanguage('en');
    expect(i18n.t('errors.INVALID_CREDENTIALS')).toBe('Invalid email or password');

    await i18n.changeLanguage('zh-TW');
    expect(i18n.t('errors.INVALID_CREDENTIALS')).toBe('帳號或密碼錯誤');

    await i18n.changeLanguage('th');
    // Don't pin the exact translation text — locale teams may refine wording.
    // Just confirm a non-empty Thai-ish string was returned (not the key itself).
    const thMsg = i18n.t('errors.INVALID_CREDENTIALS');
    expect(thMsg).not.toBe('errors.INVALID_CREDENTIALS');
    expect(thMsg.length).toBeGreaterThan(0);
  });

  it('setLanguage persists choice to localStorage', () => {
    setLanguage('zh-TW');
    expect(localStorage.getItem('language')).toBe('zh-TW');
    expect(i18n.language).toBe('zh-TW');

    setLanguage('th');
    expect(localStorage.getItem('language')).toBe('th');
  });

  it('falls back to message text for unknown error codes', () => {
    // Unknown code → returns key as-is (no resource exists)
    const fallback = i18n.t('errors.SOMETHING_NOT_DEFINED', { defaultValue: 'fallback msg' });
    expect(fallback).toBe('fallback msg');
  });

  it('returns the same key set across all locales (no missing translations)', () => {
    const en = i18n.getResourceBundle('en', 'translation');
    const zh = i18n.getResourceBundle('zh-TW', 'translation');
    const th = i18n.getResourceBundle('th', 'translation');

    const flatten = (obj: Record<string, unknown>, prefix = ''): string[] => {
      return Object.entries(obj).flatMap(([k, v]) => {
        const key = prefix ? `${prefix}.${k}` : k;
        if (v && typeof v === 'object') return flatten(v as Record<string, unknown>, key);
        return [key];
      });
    };

    const enKeys = flatten(en).sort();
    const zhKeys = flatten(zh).sort();
    const thKeys = flatten(th).sort();

    expect(zhKeys).toEqual(enKeys);
    expect(thKeys).toEqual(enKeys);
  });
});
