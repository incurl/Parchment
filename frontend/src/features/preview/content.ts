/**
 * The printed content: the address blocks, the return address, the stamp
 * and the letter lines.
 *
 * These helpers describe what is printed, not where the fold is. The
 * fold geometry arrives in the scene; this module reads it.
 */

import type { Letter } from '../../bindings';
import { INK_COLOUR, PRINT_FONT } from './geometry';

/** Draws text in the current frame, aligned left or right at the origin. */
export function textAt(
  ctx: CanvasRenderingContext2D,
  content: string,
  sizeMM: number,
  colour: string,
  bold: boolean,
  align: 'left' | 'right' = 'left',
): void {
  ctx.font = `${bold ? '600' : '400'} ${sizeMM}px ${PRINT_FONT}`;
  ctx.fillStyle = colour;
  ctx.textBaseline = 'alphabetic';
  const width = ctx.measureText(content).width;
  ctx.fillText(content, align === 'right' ? -width : 0, 0);
}

/** Builds the "City, ST 12345" line. */
export function joinCityLine(city: string, state: string, zip: string): string {
  let line = city.trim();
  if (state.trim() !== '') {
    line = line === '' ? state.trim() : `${line}, ${state.trim()}`;
  }
  if (zip.trim() !== '') {
    line = line === '' ? zip.trim() : `${line}  ${zip.trim()}`;
  }
  return line;
}

/** The recipient block's lines, in reading order. */
export function addressLines(letter: Letter): { text: string; bold: boolean }[] {
  return [
    { text: letter.recipient_name, bold: true },
    { text: letter.recipient_street, bold: false },
    {
      text: joinCityLine(
        letter.recipient_city,
        letter.recipient_state,
        letter.recipient_zip,
      ),
      bold: false,
    },
  ].filter((line) => line.text.trim() !== '');
}

/** The return address's lines. */
export function returnLines(letter: Letter): string[] {
  return [
    letter.return_name,
    letter.return_street,
    joinCityLine(letter.return_city, letter.return_state, letter.return_zip),
  ].filter((line) => line.trim() !== '');
}

/** The ink colour used for every text run. */
export const TEXT_COLOUR = INK_COLOUR;
