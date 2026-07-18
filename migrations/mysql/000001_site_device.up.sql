CREATE TABLE cities (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(32) NOT NULL,
  name VARCHAR(64) NOT NULL,
  province VARCHAR(64) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uk_cities_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE sites (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_no VARCHAR(32) NOT NULL,
  city_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  address VARCHAR(255) NOT NULL,
  latitude DECIMAL(10,7) NOT NULL,
  longitude DECIMAL(10,7) NOT NULL,
  open_time TIME NOT NULL,
  close_time TIME NOT NULL,
  contact_phone VARCHAR(32) NOT NULL,
  service_status ENUM('ACTIVE','SUSPENDED','CLOSED') NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uk_sites_site_no (site_no),
  KEY idx_sites_city_status (city_id, service_status),
  KEY idx_sites_coordinates (latitude, longitude),
  CONSTRAINT fk_sites_city FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE locker_devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_no VARCHAR(64) NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  protocol_type VARCHAR(32) NOT NULL DEFAULT 'SIMULATOR',
  network_status ENUM('ONLINE','OFFLINE') NOT NULL DEFAULT 'OFFLINE',
  operational_status ENUM('ACTIVE','MAINTENANCE','DISABLED') NOT NULL DEFAULT 'ACTIVE',
  last_heartbeat_at DATETIME(6) NULL,
  firmware_version VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uk_locker_devices_device_no (device_no),
  KEY idx_locker_devices_site_status (site_id, network_status, operational_status),
  CONSTRAINT fk_locker_devices_site FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE locker_cells (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL,
  cell_no VARCHAR(32) NOT NULL,
  size ENUM('SMALL','MEDIUM','LARGE') NOT NULL,
  occupancy_status ENUM('IDLE','LOCKED','OCCUPIED','DISABLED') NOT NULL DEFAULT 'IDLE',
  door_status ENUM('CLOSED','OPEN','UNKNOWN') NOT NULL DEFAULT 'CLOSED',
  reservation_key VARCHAR(128) NULL,
  lock_expires_at DATETIME(6) NULL,
  current_order_id BIGINT UNSIGNED NULL,
  version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uk_locker_cells_device_cell (device_id, cell_no),
  UNIQUE KEY uk_locker_cells_reservation_key (reservation_key),
  KEY idx_locker_cells_allocation (device_id, size, occupancy_status, lock_expires_at),
  CONSTRAINT fk_locker_cells_device FOREIGN KEY (device_id) REFERENCES locker_devices(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE device_commands (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  command_no CHAR(26) NOT NULL,
  device_id BIGINT UNSIGNED NOT NULL,
  action ENUM('OPEN_DOOR','QUERY_STATUS') NOT NULL,
  payload_json JSON NOT NULL,
  idempotency_key VARCHAR(128) NOT NULL,
  status ENUM('PENDING','RUNNING','SUCCEEDED','FAILED','TIMED_OUT','EXPIRED') NOT NULL DEFAULT 'PENDING',
  expires_at DATETIME(6) NOT NULL,
  attempt_count INT UNSIGNED NOT NULL DEFAULT 0,
  result_json JSON NULL,
  error_code VARCHAR(64) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY uk_device_commands_command_no (command_no),
  UNIQUE KEY uk_device_commands_idempotency_key (idempotency_key),
  KEY idx_device_commands_device_status (device_id, status, created_at),
  CONSTRAINT fk_device_commands_device FOREIGN KEY (device_id) REFERENCES locker_devices(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
