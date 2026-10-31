-- A zero handled_seq is not an acknowledgement: -1 means never acknowledged.
ALTER TABLE device ALTER COLUMN handled_seq SET DEFAULT -1;
UPDATE device SET handled_seq=-1 WHERE handled_seq=0 AND boot_seq=0 AND person_handled=0;
