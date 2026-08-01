-- Allow the application user and root to connect from any host.
--
-- In CI the `make migrate-up` step runs on the GitHub Actions runner host and
-- reaches the MySQL container through the published 127.0.0.1:3306 port mapping.
-- Docker then forwards the connection with the bridge gateway IP (e.g.
-- 172.18.0.1) as the client source, so MySQL must have an account whose host
-- matches '%'. The official image does not reliably guarantee `locker`@'%',
-- which surfaces as: Host '172.18.0.1' is not allowed to connect.
--
-- This script is idempotent (CREATE USER IF NOT EXISTS) and only runs when the
-- data directory is initialised, so it is safe for both local and CI use.

CREATE USER IF NOT EXISTS 'locker'@'%' IDENTIFIED BY 'change-this-local-password';
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER, REFERENCES, EXECUTE
  ON `smart_parcel_locker`.* TO 'locker'@'%';

CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY 'change-this-root-password';
GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION;

FLUSH PRIVILEGES;
