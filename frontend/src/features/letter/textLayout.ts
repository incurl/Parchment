/**
 * Text layout helpers for the letter body.
 *
 * They are pure functions over a measurement callback so they can be
 * unit tested without a canvas. The renderer passes
 * `(text) => ctx.measureText(text).width`, so the wrap used on screen and
 * in the 3D texture is exactly the wrap used to decide whether the letter
 * fits.
 */

/** Measures a string's width in millimetres. */
export type Measure = (text: string) => number;

/** A wrapped line together with its width. */
export interface Line {
  text: string;
  width: number;
}

/**
 * wrapText breaks the text into lines no wider than maxWidth.
 *
 * Author line breaks are preserved: an empty paragraph becomes an empty
 * line, which keeps the spacing the writer intended.
 */
export function wrapText(text: string, maxWidth: number, measure: Measure): Line[] {
  const lines: Line[] = [];

  for (const paragraph of text.replace(/\r\n/g, '\n').split('\n')) {
    const words = paragraph.trim() === '' ? [] : paragraph.trim().split(/\s+/);
    if (words.length === 0) {
      lines.push({ text: '', width: 0 });
      continue;
    }

    let current = words[0];
    for (const word of words.slice(1)) {
      const candidate = `${current} ${word}`;
      if (measure(candidate) <= maxWidth) {
        current = candidate;
        continue;
      }
      lines.push({ text: current, width: measure(current) });
      current = word;
    }
    lines.push({ text: current, width: measure(current) });
  }

  return lines;
}

/** The result of laying the letter out inside its safe area. */
export interface Fit {
  /** Whether every line fits the height and the width. */
  fits: boolean;
  /** The wrapped lines. */
  lines: Line[];
  /** The height the block occupies, in millimetres. */
  height: number;
  /** The number of lines that are wider than the safe area. */
  overflowLines: number;
}

/**
 * letterFit lays the letter out and reports whether it stays inside the
 * page area that the fold leaves uncovered.
 *
 * Text outside that area would run into the fold margin: it is creased by
 * the flaps and partly hidden, which is the warning the editor shows.
 */
export function letterFit(
  text: string,
  maxWidth: number,
  maxHeight: number,
  lineHeight: number,
  measure: Measure,
): Fit {
  const lines = wrapText(text, maxWidth, measure);
  const height = lines.length * lineHeight;
  const overflowLines = lines.filter((line) => line.width > maxWidth + 0.01).length;

  return {
    fits: height <= maxHeight && overflowLines === 0,
    lines,
    height,
    overflowLines,
  };
}

/**
 * estimateMeasure returns a measure callback that assumes an average
 * glyph width.
 *
 * It is the fallback for the warning badge when no canvas is available,
 * and it keeps the unit tests independent of a browser. The renderer
 * always uses the real measurement.
 */
export function estimateMeasure(fontSizeMM: number): Measure {
  return (text: string) => text.length * fontSizeMM * 0.52;
}
