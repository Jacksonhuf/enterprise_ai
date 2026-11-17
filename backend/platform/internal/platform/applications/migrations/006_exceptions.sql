CREATE TABLE cognara_fg_exceptions (
 tenant_id VARBINARY(64) NOT NULL,
 exception_id VARBINARY(512) NOT NULL,
 ordinal BIGINT UNSIGNED NOT NULL,
 status VARBINARY(32) NOT NULL,
 request_id VARBINARY(512) NOT NULL,
 metadata JSON NOT NULL,
 raw_payload LONGBLOB NOT NULL,
 PRIMARY KEY(tenant_id,exception_id),
 UNIQUE KEY exception_global_id(exception_id),
 KEY exception_tenant_status(tenant_id,status,ordinal),
 KEY exception_request(tenant_id,request_id),
 CONSTRAINT exception_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;