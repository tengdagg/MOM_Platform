-- Rollback: Drop all tables in reverse dependency order

SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `plugin_states`;
DROP TABLE IF EXISTS `ai_skill_definitions`;
DROP TABLE IF EXISTS `ai_chat_messages`;
DROP TABLE IF EXISTS `ai_chat_sessions`;
DROP TABLE IF EXISTS `ai_model_configs`;
DROP TABLE IF EXISTS `system_config`;
DROP TABLE IF EXISTS `network_devices`;
DROP TABLE IF EXISTS `sys_asset_authorization`;
DROP TABLE IF EXISTS `sys_ldap_config`;
DROP TABLE IF EXISTS `domain_check_histories`;
DROP TABLE IF EXISTS `alert_logs`;
DROP TABLE IF EXISTS `alert_receiver_channels`;
DROP TABLE IF EXISTS `alert_receivers`;
DROP TABLE IF EXISTS `alert_channels`;
DROP TABLE IF EXISTS `alert_configs`;
DROP TABLE IF EXISTS `domain_monitors`;
DROP TABLE IF EXISTS `k8s_terminal_sessions`;
DROP TABLE IF EXISTS `k8s_cluster_inspections`;
DROP TABLE IF EXISTS `k8s_user_role_bindings`;
DROP TABLE IF EXISTS `k8s_user_kube_configs`;
DROP TABLE IF EXISTS `k8s_clusters`;
DROP TABLE IF EXISTS `k8s_helm_repos`;
DROP TABLE IF EXISTS `ansible_tasks`;
DROP TABLE IF EXISTS `job_tasks`;
DROP TABLE IF EXISTS `job_templates`;
DROP TABLE IF EXISTS `ssh_terminal_sessions`;
DROP TABLE IF EXISTS `sys_role_asset_permission`;
DROP TABLE IF EXISTS `cloud_accounts`;
DROP TABLE IF EXISTS `hosts`;
DROP TABLE IF EXISTS `credentials`;
DROP TABLE IF EXISTS `asset_group`;
DROP TABLE IF EXISTS `sys_data_log`;
DROP TABLE IF EXISTS `sys_login_log`;
DROP TABLE IF EXISTS `sys_operation_log`;
DROP TABLE IF EXISTS `sys_role_menu`;
DROP TABLE IF EXISTS `sys_user_position`;
DROP TABLE IF EXISTS `sys_user_role`;
DROP TABLE IF EXISTS `sys_position`;
DROP TABLE IF EXISTS `sys_menu`;
DROP TABLE IF EXISTS `sys_department`;
DROP TABLE IF EXISTS `sys_role`;
DROP TABLE IF EXISTS `sys_user`;

SET FOREIGN_KEY_CHECKS = 1;
