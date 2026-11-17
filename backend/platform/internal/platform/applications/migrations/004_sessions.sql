CREATE TABLE cognara_fg_sessions (
 tenant_id VARBINARY(64) NOT NULL,
 session_id VARBINARY(512) NOT NULL,
 app_id VARBINARY(512) NOT NULL,
 owner_id VARBINARY(512) NOT NULL,
 title_customized BOOLEAN NOT NULL,
 metadata JSON NOT NULL,
 raw_payload LONGBLOB NOT NULL,
 PRIMARY KEY(tenant_id,session_id),
 UNIQUE KEY session_global_id(session_id),
 KEY session_owner(tenant_id,owner_id,app_id),
 KEY session_app(tenant_id,app_id),
 CONSTRAINT session_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
CREATE TABLE cognara_fg_session_messages (
 tenant_id VARBINARY(64) NOT NULL,
 session_id VARBINARY(512) NOT NULL,
 message_id VARBINARY(512) NOT NULL,
 ordinal INT UNSIGNED NOT NULL,
 metadata JSON NOT NULL,
 raw_payload LONGBLOB NOT NULL,
 PRIMARY KEY(tenant_id,session_id,message_id),
 UNIQUE KEY session_message_order(tenant_id,session_id,ordinal),
 KEY session_message_global(message_id),
 CONSTRAINT session_message_metadata_object CHECK(JSON_TYPE(metadata)='OBJECT')
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;