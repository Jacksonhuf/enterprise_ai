CREATE TABLE cognara_fg_settings (
 tenant_id VARBINARY(64) NOT NULL,
 setting_key VARBINARY(512) NOT NULL,
 owner_id VARBINARY(512) NOT NULL,
 section VARBINARY(128) NOT NULL,
 revision VARBINARY(512) NOT NULL,
 metadata JSON NOT NULL,
 raw_payload JSON NOT NULL,
 PRIMARY KEY(tenant_id,setting_key),
 UNIQUE KEY setting_key_global(setting_key),
 KEY setting_owner(tenant_id,owner_id,section),
 CONSTRAINT setting_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT'),
 CONSTRAINT setting_payload_json CHECK(JSON_TYPE(raw_payload)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;