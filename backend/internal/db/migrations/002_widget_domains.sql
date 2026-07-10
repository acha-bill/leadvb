CREATE TABLE IF NOT EXISTS widget_domains (
  domain VARCHAR(190) PRIMARY KEY,
  account_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_widget_domains_account (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
