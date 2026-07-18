INSERT INTO cities (id, code, name, province, enabled)
VALUES (1, '310100', '上海市', '上海市', TRUE);

INSERT INTO sites (id, site_no, city_id, name, address, latitude, longitude, open_time, close_time, contact_phone, service_status)
VALUES
  (1, 'SITE-SH-001', 1, '人民广场寄存点', '上海市黄浦区人民大道 100 号', 31.2304000, 121.4737000, '08:00:00', '22:00:00', '13800000001', 'ACTIVE'),
  (2, 'SITE-SH-002', 1, '南京东路寄存点', '上海市黄浦区南京东路 200 号', 31.2361000, 121.4802000, '09:00:00', '21:00:00', '13800000002', 'ACTIVE');

INSERT INTO locker_devices (id, device_no, site_id, protocol_type, network_status, operational_status, last_heartbeat_at, firmware_version)
VALUES
  (1, 'DEV-SH-001', 1, 'SIMULATOR', 'ONLINE', 'ACTIVE', UTC_TIMESTAMP(6), 'sim-1.0.0'),
  (2, 'DEV-SH-002', 2, 'SIMULATOR', 'ONLINE', 'ACTIVE', UTC_TIMESTAMP(6), 'sim-1.0.0');

INSERT INTO locker_cells (device_id, cell_no, size, occupancy_status, door_status)
VALUES
  (1, 'A01', 'SMALL', 'IDLE', 'CLOSED'),
  (1, 'A02', 'SMALL', 'IDLE', 'CLOSED'),
  (1, 'B01', 'MEDIUM', 'IDLE', 'CLOSED'),
  (1, 'C01', 'LARGE', 'IDLE', 'CLOSED'),
  (2, 'A01', 'SMALL', 'IDLE', 'CLOSED'),
  (2, 'B01', 'MEDIUM', 'IDLE', 'CLOSED'),
  (2, 'B02', 'MEDIUM', 'IDLE', 'CLOSED'),
  (2, 'C01', 'LARGE', 'IDLE', 'CLOSED');
