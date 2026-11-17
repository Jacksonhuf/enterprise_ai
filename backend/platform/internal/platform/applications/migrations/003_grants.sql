CREATE TABLE cognara_fg_grants (
 tenant_id VARBINARY(64) NOT NULL,
 grant_id VARBINARY(512) NOT NULL,
 subject_type VARBINARY(64) NOT NULL,
 subject_id VARBINARY(512) NOT NULL,
 resource_type VARBINARY(64) NOT NULL,
 resource_id VARBINARY(512) NOT NULL,
 action VARBINARY(128) NOT NULL,
 scope VARBINARY(128) NOT NULL,
 active BOOLEAN NOT NULL,
 metadata JSON NOT NULL,
 raw_payload LONGBLOB NOT NULL,
 PRIMARY KEY(tenant_id,grant_id),
 UNIQUE KEY grant_global_id(grant_id),
 KEY grant_subject(tenant_id,subject_type,subject_id,resource_type,resource_id,action,scope),
 KEY grant_resource(tenant_id,resource_type,resource_id,action,scope),
 CONSTRAINT grant_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;