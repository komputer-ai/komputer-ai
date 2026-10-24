import { describe, expect, it } from 'vitest';
import { summarizeBulkArchive } from './bulk-archive';

const a = (name: string) => ({ name, namespace: 'default' });
const ok = { status: 'fulfilled', value: {} } as const;
const fail = (msg: string) => ({ status: 'rejected', reason: new Error(msg) }) as const;

describe('summarizeBulkArchive', () => {
  it('has no message when every archive succeeded', () => {
    const out = summarizeBulkArchive([a('x'), a('y')], [ok, ok]);
    expect(out.archived).toBe(2);
    expect(out.failedKeys).toEqual([]);
    expect(out.message).toBeNull();
  });

  it('counts the archived ones and groups the failures by reason', () => {
    const out = summarizeBulkArchive(
      [a('x'), a('busy1'), a('squad1'), a('busy2'), a('y')],
      [ok, fail('agent is running a task; cancel it first'), fail('agent is in a squad; remove it from the squad first'),
        fail('agent is running a task; cancel it first'), ok],
    );
    expect(out.archived).toBe(2);
    expect(out.failedKeys).toEqual(['default/busy1', 'default/squad1', 'default/busy2']);
    expect(out.message).toBe(
      'Archived 2 of 5. Not archived: busy1, busy2 (agent is running a task; cancel it first); squad1 (agent is in a squad; remove it from the squad first)',
    );
  });

  it('explains a failure without a message', () => {
    const out = summarizeBulkArchive([a('x')], [{ status: 'rejected', reason: 'boom' }]);
    expect(out.message).toBe('Archived 0 of 1. Not archived: x (boom)');
  });
});
