import test from 'node:test';
import assert from 'node:assert/strict';
import { parsePort } from '../src/settings.js';
test('port validation rejects partial, decimal and out-of-range values', () => {
  for (const value of ['', '1x', '-1', '1.5', '65536', 'Infinity']) assert.throws(() => parsePort(value));
  assert.equal(parsePort('0'), 0);
  assert.equal(parsePort('65535'), 65535);
});
