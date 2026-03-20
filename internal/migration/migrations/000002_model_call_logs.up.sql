CREATE TABLE IF NOT EXISTS `ai_model_call_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `session_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '会话ID（仅作关联，删除会话后统计仍保留）',
  `user_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '用户ID',
  `model_id` bigint unsigned NOT NULL DEFAULT 0 COMMENT '模型ID',
  `model_name` varchar(100) NOT NULL COMMENT '模型名称快照',
  `provider` varchar(50) DEFAULT '' COMMENT '模型提供商快照',
  `prompt_tokens` int NOT NULL DEFAULT 0 COMMENT '输入 tokens',
  `completion_tokens` int NOT NULL DEFAULT 0 COMMENT '输出 tokens',
  `total_tokens` int NOT NULL DEFAULT 0 COMMENT '总 tokens',
  `created_at` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '调用时间',
  PRIMARY KEY (`id`),
  KEY `idx_ai_model_call_logs_created_at` (`created_at`),
  KEY `idx_ai_model_call_logs_model_id` (`model_id`),
  KEY `idx_ai_model_call_logs_user_id` (`user_id`),
  KEY `idx_ai_model_call_logs_session_id` (`session_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 模型调用日志';

INSERT INTO `ai_model_call_logs` (
  `session_id`,
  `user_id`,
  `model_id`,
  `model_name`,
  `provider`,
  `prompt_tokens`,
  `completion_tokens`,
  `total_tokens`,
  `created_at`
)
SELECT
  COALESCE(s.id, 0) AS session_id,
  COALESCE(s.user_id, 0) AS user_id,
  COALESCE(s.model_id, 0) AS model_id,
  COALESCE(NULLIF(mc.name, ''), NULLIF(default_mc.name, ''), '未知模型') AS model_name,
  COALESCE(NULLIF(mc.provider, ''), NULLIF(default_mc.provider, ''), '') AS provider,
  0 AS prompt_tokens,
  0 AS completion_tokens,
  COALESCE(m.tokens_used, 0) AS total_tokens,
  m.created_at
FROM `ai_chat_messages` m
LEFT JOIN `ai_chat_sessions` s ON s.id = m.session_id
LEFT JOIN `ai_model_configs` mc ON mc.id = s.model_id
LEFT JOIN `ai_model_configs` default_mc ON default_mc.is_default = 1 AND default_mc.status = 1 AND default_mc.deleted_at IS NULL
WHERE m.role = 'assistant';
