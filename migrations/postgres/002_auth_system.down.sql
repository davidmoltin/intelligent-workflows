-- Drop tables in reverse order (respecting foreign key constraints)
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS rate_limits;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
