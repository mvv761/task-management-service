CREATE TABLE IF NOT EXISTS users (
                                     id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                                     email         VARCHAR(255) NOT NULL UNIQUE,
                                     password_hash VARCHAR(255) NOT NULL,
                                     name          VARCHAR(255) NOT NULL,
                                     created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS teams (
                                     id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                                     name       VARCHAR(255) NOT NULL,
                                     created_by BIGINT UNSIGNED NOT NULL,
                                     created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

                                     FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS team_members (
                                            team_id   BIGINT UNSIGNED NOT NULL,
                                            user_id   BIGINT UNSIGNED NOT NULL,
                                            role      VARCHAR(20) NOT NULL,

                                            PRIMARY KEY (team_id, user_id),

                                            FOREIGN KEY (team_id) REFERENCES teams(id),
                                            FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS tasks (
                                     id          BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                                     team_id     BIGINT UNSIGNED NOT NULL,
                                     title       VARCHAR(255) NOT NULL,
                                     description TEXT,
                                     status      VARCHAR(20) NOT NULL DEFAULT 'todo',
                                     created_by  BIGINT UNSIGNED NOT NULL,
                                     assignee_id BIGINT UNSIGNED NULL,
                                     created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                     updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                     closed_at   TIMESTAMP NULL,
                                     version     INT UNSIGNED NOT NULL DEFAULT 1,

                                     FOREIGN KEY (team_id) REFERENCES teams(id),
                                     FOREIGN KEY (created_by) REFERENCES users(id),
                                     FOREIGN KEY (assignee_id) REFERENCES users(id),

                                     INDEX idx_tasks_team_id (team_id),
                                     INDEX idx_tasks_status (status),
                                     INDEX idx_tasks_assignee_id (assignee_id),
                                     INDEX idx_tasks_created_at (created_at),
                                     INDEX idx_tasks_team_status_created (team_id, status, created_at),
                                     INDEX idx_tasks_team_assignee_created (team_id, assignee_id, created_at)
);

CREATE TABLE IF NOT EXISTS task_history (
                                            id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                                            task_id    BIGINT UNSIGNED NOT NULL,
                                            changed_by BIGINT UNSIGNED NOT NULL,
                                            changes    JSON NOT NULL,
                                            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

                                            FOREIGN KEY (task_id) REFERENCES tasks(id),
                                            FOREIGN KEY (changed_by) REFERENCES users(id),

                                            INDEX idx_history_task (task_id, created_at)
);

CREATE TABLE IF NOT EXISTS task_comments (
                                             id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                                             task_id    BIGINT UNSIGNED NOT NULL,
                                             user_id    BIGINT UNSIGNED NOT NULL,
                                             content    TEXT NOT NULL,
                                             created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

                                             FOREIGN KEY (task_id) REFERENCES tasks(id),
                                             FOREIGN KEY (user_id) REFERENCES users(id),

                                             INDEX idx_comments_task (task_id, created_at)
);