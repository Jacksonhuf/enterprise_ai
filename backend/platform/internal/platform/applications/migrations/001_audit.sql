CREATE TABLE IF NOT EXISTS cognara_fg_audit (
 tenant_id VARBINARY(64) NOT NULL,
 event_id VARBINARY(512) NOT NULL,
 sequence_no BIGINT UNSIGNED NOT NULL,
 occurred_at DATETIME(6) NULL,
 actor_id VARCHAR(255) NOT NULL,
 action VARCHAR(255) NOT NULL,
 resource_id VARCHAR(255) NOT NULL,
 payload JSON NOT NULL CHECK(JSON_TYPE(payload)='OBJECT'),
 PRIMARY KEY(tenant_id,event_id),
 UNIQUE KEY audit_sequence(sequence_no),
 KEY audit_tenant_sequence(tenant_id,sequence_no)
) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4
