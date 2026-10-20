-- Добавляет индексы tasks, если их ещё нет (для БД со старым 000001).

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_team_id'
);
SET @sql := IF(@exist = 0, 'ALTER TABLE tasks ADD INDEX idx_tasks_team_id (team_id)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_status'
);
SET @sql := IF(@exist = 0, 'ALTER TABLE tasks ADD INDEX idx_tasks_status (status)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_assignee_id'
);
SET @sql := IF(@exist = 0, 'ALTER TABLE tasks ADD INDEX idx_tasks_assignee_id (assignee_id)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_created_at'
);
SET @sql := IF(@exist = 0, 'ALTER TABLE tasks ADD INDEX idx_tasks_created_at (created_at)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_team_status_created'
);
SET @sql := IF(
    @exist = 0,
    'ALTER TABLE tasks ADD INDEX idx_tasks_team_status_created (team_id, status, created_at)',
    'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_team_assignee_created'
);
SET @sql := IF(
    @exist = 0,
    'ALTER TABLE tasks ADD INDEX idx_tasks_team_assignee_created (team_id, assignee_id, created_at)',
    'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_team_status'
);
SET @sql := IF(@exist > 0, 'ALTER TABLE tasks DROP INDEX idx_tasks_team_status', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @exist := (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE() AND table_name = 'tasks' AND index_name = 'idx_tasks_assignee'
);
SET @sql := IF(@exist > 0, 'ALTER TABLE tasks DROP INDEX idx_tasks_assignee', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
