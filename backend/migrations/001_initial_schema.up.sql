CREATE TABLE IF NOT EXISTS tenants (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    name VARCHAR(120) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS users (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    name VARCHAR(120) NOT NULL,
    email VARCHAR(160) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    INDEX idx_users_tenant_id (tenant_id),
    INDEX idx_users_role (role)
);

CREATE TABLE IF NOT EXISTS channels (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    `key` VARCHAR(64) NOT NULL,
    name VARCHAR(120) NOT NULL,
    CONSTRAINT fk_channels_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    UNIQUE KEY uk_channels_tenant_key (tenant_id, `key`),
    INDEX idx_channels_tenant_id (tenant_id)
);

CREATE TABLE IF NOT EXISTS customers (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    external_id VARCHAR(160) NOT NULL,
    name VARCHAR(120) NOT NULL,
    CONSTRAINT fk_customers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    UNIQUE KEY uk_customers_tenant_external (tenant_id, external_id),
    INDEX idx_customers_tenant_id (tenant_id)
);

CREATE TABLE IF NOT EXISTS conversations (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    customer_id CHAR(36) NOT NULL,
    channel_id CHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL,
    assigned_agent_id CHAR(36) NULL,
    CONSTRAINT fk_conversations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_conversations_customer FOREIGN KEY (customer_id) REFERENCES customers(id),
    CONSTRAINT fk_conversations_channel FOREIGN KEY (channel_id) REFERENCES channels(id),
    CONSTRAINT fk_conversations_assigned_agent FOREIGN KEY (assigned_agent_id) REFERENCES users(id),
    INDEX idx_conversations_tenant_status (tenant_id, status),
    INDEX idx_conversations_tenant_agent (tenant_id, assigned_agent_id),
    INDEX idx_conversations_customer_channel (customer_id, channel_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    conversation_id CHAR(36) NOT NULL,
    sender_type VARCHAR(32) NOT NULL,
    sender_id CHAR(36) NULL,
    message TEXT NOT NULL,
    CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    INDEX idx_messages_conversation (conversation_id, created_at),
    INDEX idx_messages_sender (sender_id)
);

CREATE TABLE IF NOT EXISTS tickets (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    conversation_id CHAR(36) NOT NULL,
    title VARCHAR(180) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    priority VARCHAR(32) NOT NULL,
    assigned_agent_id CHAR(36) NULL,
    CONSTRAINT fk_tickets_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_tickets_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    CONSTRAINT fk_tickets_assigned_agent FOREIGN KEY (assigned_agent_id) REFERENCES users(id),
    UNIQUE KEY uk_tickets_conversation (conversation_id),
    INDEX idx_tickets_tenant_status (tenant_id, status),
    INDEX idx_tickets_tenant_agent (tenant_id, assigned_agent_id)
);

CREATE TABLE IF NOT EXISTS activity_logs (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    entity_type VARCHAR(120) NOT NULL,
    entity_id CHAR(36) NOT NULL,
    payload LONGTEXT NOT NULL,
    CONSTRAINT fk_activity_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    INDEX idx_activity_logs_tenant_created (tenant_id, created_at),
    INDEX idx_activity_logs_entity (entity_type, entity_id)
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id CHAR(36) PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    tenant_id CHAR(36) NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    entity_type VARCHAR(120) NOT NULL,
    entity_id CHAR(36) NOT NULL,
    routing_key VARCHAR(160) NOT NULL,
    payload LONGTEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    published_at DATETIME NULL,
    failed_at DATETIME NULL,
    CONSTRAINT fk_outbox_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    INDEX idx_outbox_events_pending (published_at, created_at),
    INDEX idx_outbox_events_entity (entity_type, entity_id)
);