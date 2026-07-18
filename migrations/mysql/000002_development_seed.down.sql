DELETE FROM locker_cells WHERE device_id IN (1, 2);
DELETE FROM locker_devices WHERE id IN (1, 2);
DELETE FROM sites WHERE id IN (1, 2);
DELETE FROM cities WHERE id = 1;
