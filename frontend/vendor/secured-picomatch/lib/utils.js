/*global navigator*/
'use strict';

const {
  REGEX_BACKSLASH,
  REGEX_REMOVE_BACKSLASH,
  REGEX_SPECIAL_CHARS,
  REGEX_SPECIAL_CHARS_GLOBAL
} = require('./constants');

exports.isObject = val => val !== null && typeof val === 'object' && !Array.isArray(val);
exports.hasRegexChars = str => REGEX_SPECIAL_CHARS.test(str);
exports.isRegexChar = str => str.length === 1 && exports.hasRegexChars(str);
exports.escapeRegex = str => str.replace(REGEX_SPECIAL_CHARS_GLOBAL, '\\$1');
exports.toPosixSlashes = str => str.replace(REGEX_BACKSLASH, '/');

exports.isWindows = () => {
  if (typeof navigator !== 'undefined' && navigator.platform) {
    const platform = navigator.platform.toLowerCase();
    return platform === 'win32' || platform === 'windows';
  }

  if (typeof process !== 'undefined' && process.platform) {
    return process.platform === 'win32';
  }

  return false;
};

// The former bracket regexp had quadratic worst-case backtracking on repeated
// escaped opening brackets. Match its replacement semantics in linear time.
exports.removeBackslashes = str => {
  if (!str.includes('\\')) return str;
  const n = str.length;
  const nextClose = new Int32Array(n + 1);
  const nextBreak = new Int32Array(n + 1);
  let close = -1;
  let lineBreak = -1;
  const isBreak = ch => ch === '\n' || ch === '\r' || ch === '\u2028' || ch === '\u2029';
  for (let i = n - 1; i >= 0; i--) {
    if (str[i] === ']' && i > 0 && str[i - 1] !== '\\') close = i;
    if (isBreak(str[i])) lineBreak = i;
    nextClose[i] = close;
    nextBreak[i] = lineBreak;
  }

  const pieces = [];
  let segment = 0;
  for (let i = 0; i < n; i++) {
    if (str[i] === '[' && i + 2 < n) {
      // Preserve complete bracket expressions, including backslashes inside.
      const end = nextClose[i + 2];
      const stop = nextBreak[i + 1];
      if (end !== -1 && (stop === -1 || stop >= end - 1)) {
        i = end;
        continue;
      }
    }
    if (str[i] !== '\\' || i + 1 === n || isBreak(str[i + 1])) continue;    pieces.push(str.slice(segment, i));
    segment = i + 1;
  }
  if (pieces.length === 0) return str;
  pieces.push(str.slice(segment));
  return pieces.join('');
};
exports.escapeLast = (input, char, lastIdx) => {
  const idx = input.lastIndexOf(char, lastIdx);
  if (idx === -1) return input;
  if (input[idx - 1] === '\\') return exports.escapeLast(input, char, idx - 1);
  return `${input.slice(0, idx)}\\${input.slice(idx)}`;
};

exports.removePrefix = (input, state = {}) => {
  let output = input;
  if (output.startsWith('./')) {
    output = output.slice(2);
    state.prefix = './';
  }
  return output;
};

exports.wrapOutput = (input, state = {}, options = {}) => {
  const prepend = options.contains ? '' : '^';
  const append = options.contains ? '' : '$';

  let output = `${prepend}(?:${input})${append}`;
  if (state.negated === true) {
    output = `(?:^(?!${output}).*$)`;
  }
  return output;
};

exports.basename = (path, { windows } = {}) => {
  const segs = path.split(windows ? /[\\/]/ : '/');
  const last = segs[segs.length - 1];

  if (last === '') {
    return segs[segs.length - 2];
  }

  return last;
};
