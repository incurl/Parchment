import { describe, expect, it } from 'vitest';

import { estimateMeasure, letterFit, wrapText } from './textLayout';

/** A measure where every character is exactly one unit wide. */
const mono: (text: string) => number = (text) => text.length;

describe('wrapText', () => {
  it('keeps short text on one line', () => {
    const lines = wrapText('hello there', 20, mono);
    expect(lines.map((line) => line.text)).toEqual(['hello there']);
  });

  it('wraps at the last word that fits', () => {
    const lines = wrapText('the quick brown fox', 10, mono);
    expect(lines.map((line) => line.text)).toEqual(['the quick', 'brown fox']);
  });

  it('preserves author line breaks and blank lines', () => {
    const lines = wrapText('one\n\ntwo', 20, mono);
    expect(lines.map((line) => line.text)).toEqual(['one', '', 'two']);
  });

  it('normalises CRLF and collapsed whitespace', () => {
    const lines = wrapText('one\r\n   two   three ', 20, mono);
    expect(lines.map((line) => line.text)).toEqual(['one', 'two three']);
  });

  it('reports each line width', () => {
    const lines = wrapText('aa bb', 20, mono);
    expect(lines[0].width).toBe(5);
  });

  it('keeps a word that is too long on a line of its own', () => {
    const lines = wrapText('a supercalifragilistic word', 5, mono);
    expect(lines.map((line) => line.text)).toContain('supercalifragilistic');
  });
});

describe('letterFit', () => {
  it('reports a short letter as fitting', () => {
    const fit = letterFit('hello\nthere', 100, 20, 5, mono);
    expect(fit.fits).toBe(true);
    expect(fit.height).toBe(10);
    expect(fit.overflowLines).toBe(0);
  });

  it('reports too much text as not fitting', () => {
    const fit = letterFit('a\nb\nc\nd', 100, 15, 5, mono);
    expect(fit.fits).toBe(false);
    expect(fit.height).toBe(20);
  });

  it('reports a line that overflows the width', () => {
    const fit = letterFit('abcdefghij', 4, 100, 5, mono);
    expect(fit.overflowLines).toBe(1);
    expect(fit.fits).toBe(false);
  });
});

describe('estimateMeasure', () => {
  it('scales with the font size', () => {
    const small = estimateMeasure(2)('hello');
    const large = estimateMeasure(4)('hello');
    expect(large).toBeCloseTo(small * 2, 5);
  });
});
