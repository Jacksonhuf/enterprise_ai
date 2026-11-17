CREATE TABLE cognara_fg_applications (
 tenant_id VARBINARY(64) NOT NULL,
 app_id VARBINARY(512) NOT NULL,
 agent_id VARBINARY(512) NOT NULL,
 owner_id VARBINARY(512) NOT NULL,
 current_release_id VARBINARY(512) NULL,
 updated_at VARCHAR(64) NOT NULL,
 metadata JSON NOT NULL,
 releases_is_null BOOLEAN NOT NULL,
 PRIMARY KEY(tenant_id,app_id),
 UNIQUE KEY application_id(app_id),
 KEY application_owner(tenant_id,owner_id,app_id),
 KEY application_current(tenant_id,app_id,current_release_id),
 CONSTRAINT application_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT'),
 CONSTRAINT application_release_shape CHECK(releases_is_null IN (0,1))
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE TABLE cognara_fg_application_drafts (
 tenant_id VARBINARY(64) NOT NULL,
 app_id VARBINARY(512) NOT NULL,
 revision VARBINARY(512) NOT NULL,
 config JSON NOT NULL,
 PRIMARY KEY(tenant_id,app_id),
 CONSTRAINT draft_application_fk FOREIGN KEY(tenant_id,app_id) REFERENCES cognara_fg_applications(tenant_id,app_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
 CONSTRAINT draft_config_object CHECK(JSON_TYPE(config)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE TABLE cognara_fg_application_releases (
 tenant_id VARBINARY(64) NOT NULL,
 app_id VARBINARY(512) NOT NULL,
 release_id VARBINARY(512) NOT NULL,
 ordinal BIGINT UNSIGNED NOT NULL,
 snapshot JSON NOT NULL,
 published_at VARCHAR(64) NOT NULL,
 published_by VARBINARY(512) NOT NULL,
 metadata JSON NOT NULL,
 PRIMARY KEY(tenant_id,app_id,release_id),
 UNIQUE KEY release_ordinal(tenant_id,app_id,ordinal),
 CONSTRAINT release_application_fk FOREIGN KEY(tenant_id,app_id) REFERENCES cognara_fg_applications(tenant_id,app_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
 CONSTRAINT release_snapshot_object CHECK(JSON_TYPE(snapshot)='OBJECT'),
 CONSTRAINT release_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE TABLE cognara_fg_application_checks (
 tenant_id VARBINARY(64) NOT NULL,
 check_id VARBINARY(512) NOT NULL,
 app_id VARBINARY(512) NOT NULL,
 draft_revision VARBINARY(512) NOT NULL,
 user_id VARBINARY(512) NOT NULL,
 expires_at VARCHAR(64) NOT NULL,
 validation_id VARBINARY(512) NOT NULL,
 metadata JSON NOT NULL,
 PRIMARY KEY(tenant_id,check_id),
 UNIQUE KEY check_id(check_id),
 KEY check_application(tenant_id,app_id,check_id),
 CONSTRAINT check_application_fk FOREIGN KEY(tenant_id,app_id) REFERENCES cognara_fg_applications(tenant_id,app_id) ON UPDATE RESTRICT ON DELETE RESTRICT,
 CONSTRAINT check_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
ALTER TABLE cognara_fg_applications ADD CONSTRAINT application_current_fk FOREIGN KEY(tenant_id,app_id,current_release_id) REFERENCES cognara_fg_application_releases(tenant_id,app_id,release_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
