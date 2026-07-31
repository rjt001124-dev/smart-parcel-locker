ALTER TABLE device_commands
  ADD COLUMN cell_no VARCHAR(32) NOT NULL DEFAULT '' AFTER device_id;
