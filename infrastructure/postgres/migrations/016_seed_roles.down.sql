-- 016_seed_roles.down.sql
DELETE FROM identity.role_permissions;
DELETE FROM identity.permissions;
DELETE FROM identity.roles;
