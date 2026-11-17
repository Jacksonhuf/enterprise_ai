CREATE TABLE cognara_fg_usage (
 tenant_id VARBINARY(64) NOT NULL,
 event_id VARBINARY(512) NOT NULL,
 ordinal BIGINT UNSIGNED NOT NULL,
 unit VARBINARY(32) NOT NULL,
 capability VARBINARY(512) NOT NULL,
 occurred_at VARBINARY(64) NOT NULL,
 metadata JSON NOT NULL,
 raw_payload LONGBLOB NOT NULL,
 PRIMARY KEY(tenant_id,event_id),
 UNIQUE KEY usage_global_id(event_id),
 KEY usage_tenant_unit(tenant_id,unit,ordinal),
 KEY usage_tenant_capability(tenant_id,capability,ordinal),
 CONSTRAINT usage_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;