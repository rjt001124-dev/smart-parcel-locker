ALTER TABLE sites
  ADD INDEX idx_sites_nearby (city_id, service_status, latitude, longitude, id);
