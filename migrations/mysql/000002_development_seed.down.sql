START TRANSACTION;

-- This rollback is only for the deterministic local fixture before dependent development data is added.
DELETE FROM locker_cells
WHERE (device_id, cell_no) IN (
  (1, 'A01'),
  (1, 'A02'),
  (1, 'B01'),
  (1, 'C01'),
  (2, 'A01'),
  (2, 'B01'),
  (2, 'B02'),
  (2, 'C01')
);
DELETE FROM locker_devices WHERE id IN (1, 2);
DELETE FROM sites WHERE id IN (1, 2);
DELETE FROM cities WHERE id = 1;

COMMIT;
